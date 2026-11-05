import { Message as MessageType, DownloadEvent, ErrorEvent } from "@/gotypes";
import React from "react";
import Message from "./Message";
import Downloading from "./Downloading";
import { ErrorMessage } from "./ErrorMessage";
import TurnSummary from "./TurnSummary";
import {
  isBrowserComputerTool,
  parseToolArgs,
  stepNameForTool,
  type TurnStep,
} from "./turnSteps";

type TurnGroup = {
  firstCallIndex: number;
  steps: TurnStep[];
  // Message indices whose tool calls are shown in the summary, not
  // as individual tool call displays.
  callMessageIndices: Set<number>;
  // Tool-role message indices whose content is shown in the
  // summary details, not as a separate tool message.
  toolMessageIndices: Set<number>;
  // User message index that started this turn, or -1 if none.
  userIndex: number;
  // Index of the first assistant message with content after the
  // last tool call in this turn, the final answer, or -1.
  finalAnswerIndex: number;
};

function computeTurnGroups(messages: MessageType[]): TurnGroup[] {
  const groups: TurnGroup[] = [];
  let current: TurnGroup | null = null;
  const consumedToolMessages = new Set<number>();
  const startTurn = (userIndex: number) => {
    current = {
      firstCallIndex: -1,
      steps: [],
      callMessageIndices: new Set<number>(),
      toolMessageIndices: new Set<number>(),
      userIndex,
      finalAnswerIndex: -1,
    };
    groups.push(current);
  };
  startTurn(-1);
  for (let i = 0; i < messages.length; i++) {
    const message = messages[i] as any;
    if (message?.role === "user") {
      startTurn(i);
      continue;
    }
    if (!current) {
      startTurn(-1);
    }
    const calls: Array<{ name: string; argsJson: string }> = [];
    const rawCalls: any[] = Array.isArray(message?.tool_calls)
      ? message.tool_calls
      : message?.tool_call
        ? [message.tool_call]
        : [];
    for (const call of rawCalls) {
      const name = call?.function?.name as string | undefined;
      if (name && isBrowserComputerTool(name)) {
        calls.push({ name, argsJson: call.function.arguments || "{}" });
      }
    }
    if (calls.length > 0) {
      if (current!.firstCallIndex === -1) {
        current!.firstCallIndex = i;
      }
      current!.callMessageIndices.add(i);
      for (const call of calls) {
        // The tool result for a call is the next unconsumed tool
        // message after it. If none follows, the step has no result.
        let result: string | null = null;
        for (let j = i + 1; j < messages.length; j++) {
          const next = messages[j] as any;
          if (next?.role !== "tool") {
            break;
          }
          if (consumedToolMessages.has(j)) {
            continue;
          }
          consumedToolMessages.add(j);
          current!.toolMessageIndices.add(j);
          const rawResult = next.tool_result;
          result =
            typeof next.content === "string" && next.content
              ? next.content
              : rawResult !== undefined && rawResult !== null
                ? typeof rawResult === "string"
                  ? rawResult
                  : JSON.stringify(rawResult, null, 2)
                : null;
          break;
        }
        current!.steps.push({
          rawName: call.name,
          stepName: stepNameForTool(call.name, parseToolArgs(call.argsJson)),
          argsJson: call.argsJson,
          result,
        });
      }
    } else if (
      current!.steps.length > 0 &&
      current!.finalAnswerIndex === -1 &&
      message?.role === "assistant" &&
      typeof message?.content === "string" &&
      message.content.trim()
    ) {
      current!.finalAnswerIndex = i;
    }
  }
  return groups.filter((group) => group.steps.length > 0);
}

export default function MessageList({
  messages,
  spacerHeight,
  isWaitingForLoad,
  isStreaming,
  downloadProgress,
  onEditMessage,
  editingMessageIndex,
  error,
  browserToolResult,
}: {
  messages: MessageType[];
  spacerHeight: number;
  isWaitingForLoad?: boolean;
  isStreaming: boolean;
  downloadProgress?: DownloadEvent;
  onEditMessage?: (content: string, index: number) => void | Promise<void>;
  editingMessageIndex?: number;
  error?: ErrorEvent | null;
  browserToolResult?: any;
}) {
  const [showDots, setShowDots] = React.useState(false);
  const isDownloadingModel = downloadProgress && !downloadProgress.done;
  const shouldShowDownload = messages.length > 0;

  React.useEffect(() => {
    let timer: number;
    if (
      (isStreaming || isWaitingForLoad) &&
      !isDownloadingModel &&
      messages.length > 0 &&
      messages[messages.length - 1]?.role === "user"
    ) {
      timer = window.setTimeout(() => {
        setShowDots(true);
      }, 750); // Wait 750ms before showing dots
    } else {
      setShowDots(false);
    }

    return () => window.clearTimeout(timer);
  }, [isStreaming, isWaitingForLoad, isDownloadingModel, messages]);

  const lastIdx = messages.length - 1;

  // Turn timing, measured in the browser, never invented. A turn
  // start is recorded only when a user message is newly appended
  // during this session. Turns already complete when the chat is
  // first loaded get no start time, and therefore no timer, rather
  // than a made-up elapsed time. Elapsed time is recorded when
  // streaming ends and the turn has a final answer.
  const turnStartRef = React.useRef<Map<number, number>>(new Map());
  const prevLengthRef = React.useRef<number | null>(null);
  const [elapsedByUser, setElapsedByUser] = React.useState<Map<number, number>>(
    new Map(),
  );
  const turnGroups = React.useMemo(() => computeTurnGroups(messages), [messages]);
  const groupByFirstCall = React.useMemo(() => {
    const map = new Map<number, TurnGroup>();
    for (const group of turnGroups) {
      map.set(group.firstCallIndex, group);
    }
    return map;
  }, [turnGroups]);
  const groupedCallMessages = React.useMemo(() => {
    const set = new Set<number>();
    for (const group of turnGroups) {
      for (const index of group.callMessageIndices) {
        set.add(index);
      }
    }
    return set;
  }, [turnGroups]);
  const groupedToolMessages = React.useMemo(() => {
    const set = new Set<number>();
    for (const group of turnGroups) {
      for (const index of group.toolMessageIndices) {
        set.add(index);
      }
    }
    return set;
  }, [turnGroups]);
  React.useEffect(() => {
    const prevLength = prevLengthRef.current;
    prevLengthRef.current = messages.length;
    if (prevLength !== null && messages.length > prevLength) {
      for (let i = prevLength; i < messages.length; i++) {
        if ((messages[i] as any)?.role === "user" && !turnStartRef.current.has(i)) {
          turnStartRef.current.set(i, Date.now());
        }
      }
    }
    if (!isStreaming) {
      let changed = false;
      const next = new Map(elapsedByUser);
      for (const group of turnGroups) {
        if (group.userIndex < 0 || group.finalAnswerIndex === -1) {
          continue;
        }
        const start = turnStartRef.current.get(group.userIndex);
        if (start === undefined || next.has(group.userIndex)) {
          continue;
        }
        next.set(
          group.userIndex,
          Math.max(0, Math.round((Date.now() - start) / 1000)),
        );
        changed = true;
      }
      if (changed) {
        setElapsedByUser(next);
      }
    }
  }, [messages, isStreaming, turnGroups, elapsedByUser]);

  // Memoize the last tool query (web_search query or web_fetch url) at each message index
  const lastToolQueries = React.useMemo(() => {
    const queries: (string | undefined)[] = [];
    let lastQuery: string | undefined = undefined;
    for (let i = 0; i < messages.length; i++) {
      const m: any = messages[i] as any;
      const toolCalls: any[] | undefined = Array.isArray(m?.tool_calls)
        ? (m.tool_calls as any[])
        : m?.tool_call
          ? [m.tool_call]
          : undefined;
      if (toolCalls && toolCalls.length > 0) {
        for (const tc of toolCalls) {
          const name = tc?.function?.name;
          if (name === "web_search" || name === "web_fetch") {
            try {
              const args = JSON.parse(tc.function.arguments || "{}");
              const candidate =
                typeof args.query === "string" && args.query.trim()
                  ? String(args.query).trim()
                  : typeof args.url === "string" && args.url.trim()
                    ? String(args.url).trim()
                    : "";
              if (candidate) lastQuery = candidate;
            } catch { /* ignored */ }
          }
        }
      }
      queries.push(lastQuery);
    }
    return queries;
  }, [messages]);

  return (
    <div
      className="mx-auto flex max-w-[768px] flex-1 flex-col px-6 pb-12 select-text"
      data-role="message-list"
    >
      {messages.map((message, idx) => {
        // Tool messages grouped into a turn summary render nothing
        // on their own; their content is in the summary details.
        if (groupedToolMessages.has(idx)) {
          return null;
        }
        const lastToolQuery = lastToolQueries[idx];
        const group = groupByFirstCall.get(idx);
        const isLastTurn = group
          ? turnGroups[turnGroups.length - 1] === group
          : false;
        return (
          <div key={`${message.created_at}-${idx}`} data-message-index={idx}>
            <Message
              message={message}
              onEditMessage={onEditMessage}
              messageIndex={idx}
              isStreaming={isStreaming && idx === lastIdx}
              isFaded={
                editingMessageIndex !== undefined && idx >= editingMessageIndex
              }
              browserToolResult={browserToolResult}
              lastToolQuery={lastToolQuery}
              hideBrowserComputerTools={groupedCallMessages.has(idx)}
            />
            {group && (
              <TurnSummary
                steps={group.steps}
                elapsedSeconds={
                  group.userIndex >= 0
                    ? elapsedByUser.get(group.userIndex) ?? null
                    : null
                }
                isWorking={isStreaming && isLastTurn}
              />
            )}
          </div>
        );
      })}

      {/* Inline error message */}
      {error && error.code !== "cloud_unauthorized" && (
        <ErrorMessage error={error} />
      )}

      {/* Indeterminate loading indicator */}
      {showDots && (
        <div className="flex items-center space-x-1.5 py-3 mt-2 self-start rounded-full px-4 min-w-0 bg-neutral-100 dark:bg-neutral-800 border-neutral-200 dark:border-neutral-700">
          <div
            className="w-1.5 h-1.5 bg-neutral-400 dark:bg-neutral-500 rounded-full opacity-0"
            style={{
              animation: "typing 1.4s infinite",
              animationDelay: "0s",
            }}
          />
          <div
            className="w-1.5 h-1.5 bg-neutral-400 dark:bg-neutral-500 rounded-full opacity-0"
            style={{
              animation: "typing 1.4s infinite",
              animationDelay: "0.15s",
            }}
          />
          <div
            className="w-1.5 h-1.5 bg-neutral-400 dark:bg-neutral-500 rounded-full opacity-0"
            style={{
              animation: "typing 1.4s infinite",
              animationDelay: "0.3s",
            }}
          />
        </div>
      )}

      {/* Downloading model */}
      {/* Only show for models larger than 1KiB */}
      {downloadProgress?.total && downloadProgress.total > 1024 && (
        <section
          className={`
          transition-all ease-out
          ${shouldShowDownload ? "duration-300" : "duration-0"}
          ${
            downloadProgress
              ? downloadProgress.done
                ? "opacity-0 -translate-y-8"
                : "opacity-100 translate-y-0"
              : "opacity-0 translate-y-4 pointer-events-none"
          }
        `}
        >
          <Downloading
            completed={downloadProgress?.completed || 0}
            total={downloadProgress?.total || 0}
          />
        </section>
      )}

      {/* Dynamic spacer to allow scrolling the last message to the top of the container */}
      <div style={{ height: `${spacerHeight}px` }} aria-hidden="true" />
    </div>
  );
}
