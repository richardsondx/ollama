import MessageList from "./MessageList";
import ChatForm from "./ChatForm";
import { FileUpload } from "./FileUpload";
import { DisplayStale } from "./DisplayStale";
import { DisplayLogin } from "./DisplayLogin";
import {
  useChat,
  useSendMessage,
  useIsStreaming,
  useIsWaitingForLoad,
  useDownloadProgress,
  useChatError,
  useShouldShowStaleDisplay,
  useDismissStaleModel,
  useCancelMessage,
} from "@/hooks/useChats";
import { useHealth } from "@/hooks/useHealth";
import { useMessageAutoscroll } from "@/hooks/useMessageAutoscroll";
import {
  useState,
  useEffect,
  useLayoutEffect,
  useRef,
  useCallback,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useSelectedModel } from "@/hooks/useSelectedModel";
import { useHasVisionCapability } from "@/hooks/useModelCapabilities";
import { Message } from "@/gotypes";

export default function Chat({ chatId }: { chatId: string }) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const chatQuery = useChat(chatId === "new" ? "" : chatId);
  const chatErrorQuery = useChatError(chatId === "new" ? "" : chatId);
  const { selectedModel } = useSelectedModel(chatId);
  const hasVisionCapability = useHasVisionCapability(selectedModel?.model);
  const shouldShowStaleDisplay = useShouldShowStaleDisplay(selectedModel);
  const dismissStaleModel = useDismissStaleModel();
  const { isHealthy } = useHealth();

  const [editingMessage, setEditingMessage] = useState<{
    content: string;
    index: number;
    originalMessage: Message;
  } | null>(null);
  const prevChatIdRef = useRef<string>(chatId);

  const chatFormCallbackRef = useRef<
    | ((
        files: Array<{ filename: string; data: Uint8Array; type?: string }>,
        errors: Array<{ filename: string; error: string }>,
      ) => void)
    | null
  >(null);

  const handleFilesReceived = useCallback(
    (
      callback: (
        files: Array<{
          filename: string;
          data: Uint8Array;
          type?: string;
        }>,
        errors: Array<{ filename: string; error: string }>,
      ) => void,
    ) => {
      chatFormCallbackRef.current = callback;
    },
    [],
  );

  const handleFilesProcessed = useCallback(
    (
      files: Array<{ filename: string; data: Uint8Array; type?: string }>,
      errors: Array<{ filename: string; error: string }> = [],
    ) => {
      chatFormCallbackRef.current?.(files, errors);
    },
    [],
  );

  const allMessages = chatQuery?.data?.chat?.messages ?? [];
  // TODO(parthsareen): will need to consolidate when used with more tools with state
  const browserToolResult = chatQuery?.data?.chat?.browser_state;
  const chatError = chatErrorQuery.data;

  const messages = allMessages;
  const isStreaming = useIsStreaming(chatId);
  const isWaitingForLoad = useIsWaitingForLoad(chatId);
  const downloadProgress = useDownloadProgress(chatId);
  const isDownloadingModel = downloadProgress && !downloadProgress.done;
  const isDisabled = !isHealthy;

  // Clear editing state when navigating to a different chat
  useEffect(() => {
    setEditingMessage(null);
  }, [chatId]);

  const sendMessageMutation = useSendMessage(chatId);
  const cancelMessage = useCancelMessage();

  // Control banner: which browser or computer tool is running in
  // the current turn, if any. Only messages after the last user
  // message belong to the current turn, so a tool call from an
  // earlier turn never keeps the banner up. The banner shows
  // only while streaming, which is the turn-active signal the
  // UI already has, and it disappears when the turn ends. The
  // Stop button reuses the existing cancel mechanism, the same
  // one the composer stop button uses. This is a banner inside
  // the Ollamapaca window only. Ollamapaca does not capture
  // the screen, so it is not a macOS "Currently Sharing"
  // indicator, and it never claims to be one.
  const controlInfo = (() => {
    if (!isStreaming) {
      return null;
    }
    for (let i = messages.length - 1; i >= 0; i--) {
      const message = messages[i] as any;
      if (message?.role === "user") {
        break;
      }
      const calls = message?.tool_calls as Array<any> | undefined;
      if (calls) {
        for (let j = calls.length - 1; j >= 0; j--) {
          const name = calls[j]?.function?.name as string | undefined;
          if (name && (name.startsWith("browser.") || name.startsWith("computer."))) {
            let target = name.startsWith("browser.") ? "Chrome" : "your computer";
            try {
              const args = JSON.parse(calls[j].function.arguments || "{}");
              if (typeof args.app === "string" && args.app) {
                target = args.app;
              }
            } catch {
              // Keep the fallback target if arguments do not parse.
            }
            return { target, name };
          }
        }
      }
      if (message?.role === "tool") {
        const toolName = (message.tool_name || message.toolName) as string | undefined;
        if (toolName && (toolName.startsWith("browser.") || toolName.startsWith("computer."))) {
          return {
            target: toolName.startsWith("browser.") ? "Chrome" : "your computer",
            name: toolName,
          };
        }
      }
    }
    return null;
  })();

  const { containerRef, handleNewUserMessage, spacerHeight } =
    useMessageAutoscroll({
      messages,
      isStreaming,
      chatId,
    });

  // Scroll to bottom only when switching to a different existing chat
  useLayoutEffect(() => {
    // Only scroll if the chatId actually changed (not just messages updating)
    if (
      prevChatIdRef.current !== chatId &&
      containerRef.current &&
      messages.length > 0 &&
      chatId !== "new"
    ) {
      // Always scroll to the bottom when opening a chat
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
    prevChatIdRef.current = chatId;
  }, [chatId, messages.length]);

  // Simplified submit handler - ChatForm handles all the attachment logic
  const handleChatFormSubmit = (
    message: string,
    options: {
      attachments?: Array<{ filename: string; data: Uint8Array }>;
      index?: number;
      webSearch?: boolean;
      fileTools?: boolean;
      think?: boolean | string;
    },
  ) => {
    // Clear any existing errors when sending a new message
    sendMessageMutation.reset();
    if (chatError) {
      clearChatError();
    }

    // Prepare attachments for backend
    const allAttachments = (options.attachments || []).map((att) => ({
      filename: att.filename,
      data: att.data.length === 0 ? new Uint8Array(0) : att.data,
    }));

    sendMessageMutation.mutate({
      message,
      attachments: allAttachments,
      index: editingMessage ? editingMessage.index : options.index,
      webSearch: options.webSearch,
      fileTools: options.fileTools,
      think: options.think,
      onChatEvent: (event) => {
        if (event.eventName === "chat_created" && event.chatId) {
          navigate({
            to: "/c/$chatId",
            params: {
              chatId: event.chatId,
            },
          });
        }
      },
    });

    // Clear edit mode after submission
    setEditingMessage(null);
    handleNewUserMessage();
  };

  const handleEditMessage = (content: string, index: number) => {
    setEditingMessage({
      content,
      index,
      originalMessage: messages[index],
    });
  };

  const handleCancelEdit = () => {
    setEditingMessage(null);
    if (chatError) {
      clearChatError();
    }
  };

  const clearChatError = () => {
    queryClient.setQueryData(
      ["chatError", chatId === "new" ? "" : chatId],
      null,
    );
  };

  const isWindows = navigator.platform.toLowerCase().includes("win");

  return chatId === "new" || chatQuery ? (
    <FileUpload
      onFilesAdded={handleFilesProcessed}
      selectedModel={selectedModel}
      hasVisionCapability={hasVisionCapability}
    >
      {chatId === "new" ? (
        <div className="flex flex-col h-screen justify-center relative">
          <div className="px-6">
            <ChatForm
              hasMessages={false}
              onSubmit={handleChatFormSubmit}
              chatId={chatId}
              autoFocus={true}
              editingMessage={editingMessage}
              onCancelEdit={handleCancelEdit}
              isDownloadingModel={isDownloadingModel}
              isDisabled={isDisabled}
              onFilesReceived={handleFilesReceived}
            />
          </div>
        </div>
      ) : (
        <main className="flex h-screen w-full flex-col relative allow-context-menu select-none">
          <section
            key={chatId} // This key forces React to recreate the element when chatId changes
            ref={containerRef}
            className={`flex-1 overflow-y-auto overscroll-contain relative min-h-0 select-none ${isWindows ? "xl:pt-4" : "xl:pt-8"}`}
          >
            <MessageList
              messages={messages}
              spacerHeight={spacerHeight}
              isWaitingForLoad={isWaitingForLoad}
              isStreaming={isStreaming}
              downloadProgress={downloadProgress}
              onEditMessage={(content: string, index: number) => {
                handleEditMessage(content, index);
              }}
              editingMessageIndex={editingMessage?.index}
              error={chatError}
              browserToolResult={browserToolResult}
            />
          </section>

          <div className="flex-shrink-0 sticky bottom-0 z-20">
            {selectedModel && shouldShowStaleDisplay && (
              <div className="pb-2">
                <DisplayStale
                  model={selectedModel}
                  onDismiss={() =>
                    dismissStaleModel(selectedModel?.model || "")
                  }
                  chatId={chatId}
                  onScrollToBottom={() => {
                    if (containerRef.current) {
                      containerRef.current.scrollTo({
                        top: containerRef.current.scrollHeight,
                        behavior: "smooth",
                      });
                    }
                  }}
                />
              </div>
            )}
            {chatError && chatError.code === "cloud_unauthorized" && (
              <div className="pb-2">
                <DisplayLogin error={chatError} />
              </div>
            )}
            {controlInfo && (
              <div className="pb-2">
                <div className="flex items-center justify-between gap-3 rounded-lg border border-neutral-300 bg-neutral-100 px-3 py-2 text-sm text-neutral-800 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100">
                  <span>
                    Controlling {controlInfo.target} - {controlInfo.name}
                  </span>
                  <button
                    type="button"
                    onClick={() => cancelMessage(chatId)}
                    className="shrink-0 rounded-full bg-black px-3 py-1 text-white dark:bg-white dark:text-black"
                  >
                    Stop
                  </button>
                </div>
              </div>
            )}
            <ChatForm
              hasMessages={messages.length > 0}
              onSubmit={handleChatFormSubmit}
              chatId={chatId}
              autoFocus={true}
              editingMessage={editingMessage}
              onCancelEdit={handleCancelEdit}
              isDisabled={isDisabled}
              isDownloadingModel={isDownloadingModel}
              onFilesReceived={handleFilesReceived}
            />
          </div>
        </main>
      )}
    </FileUpload>
  ) : (
    <div>Loading...</div>
  );
}
