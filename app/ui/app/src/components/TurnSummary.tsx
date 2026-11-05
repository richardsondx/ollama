import React from "react";
import type { TurnStep } from "./turnSteps";

function TurnStepRow({ step }: { step: TurnStep }) {
  const [showDetails, setShowDetails] = React.useState(false);
  return (
    <div className="py-1">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-neutral-800 dark:text-neutral-200">
          {step.stepName}
        </span>
        <button
          type="button"
          onClick={() => setShowDetails(!showDetails)}
          className="shrink-0 cursor-pointer text-xs text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-200"
        >
          {showDetails ? "Hide details" : "Details"}
        </button>
      </div>
      {showDetails && (
        <div className="mt-1 space-y-1 text-xs text-neutral-500 dark:text-neutral-400">
          <div>
            Tool: <span className="font-mono">{step.rawName}</span>
          </div>
          <pre className="max-h-40 overflow-auto whitespace-pre-wrap rounded-md bg-neutral-100 p-2 dark:bg-neutral-800">
            <code>{step.argsJson}</code>
          </pre>
          {step.result !== null && (
            <pre className="max-h-40 overflow-auto whitespace-pre-wrap rounded-md bg-neutral-100 p-2 dark:bg-neutral-800">
              <code>{step.result}</code>
            </pre>
          )}
        </div>
      )}
    </div>
  );
}

// One collapsible summary for a browser or computer turn. Step names
// are visible by default. Raw tool names, argument JSON, and result
// JSON are hidden behind each step's Details toggle. There is no
// live window thumbnail here: step 1 has no screen capture, and
// this component never claims to show one.
export default function TurnSummary({
  steps,
  elapsedSeconds,
  isWorking,
}: {
  steps: TurnStep[];
  elapsedSeconds: number | null;
  isWorking: boolean;
}) {
  const [expanded, setExpanded] = React.useState(true);
  const header = isWorking
    ? "Working"
    : elapsedSeconds !== null
      ? `Worked for ${elapsedSeconds}s`
      : "Browser and computer steps";
  return (
    <div className="mb-4 rounded-lg border border-neutral-200 px-3 py-2 dark:border-neutral-700">
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full cursor-pointer items-center justify-between gap-3 text-left text-sm text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-neutral-100"
      >
        <span>{header}</span>
        <span aria-hidden="true">{expanded ? "v" : ">"}</span>
      </button>
      {expanded && (
        <div className="mt-1 divide-y divide-neutral-100 dark:divide-neutral-800">
          {steps.map((step, index) => (
            <TurnStepRow
              key={`${step.rawName}-${step.argsJson}-${index}`}
              step={step}
            />
          ))}
        </div>
      )}
    </div>
  );
}
