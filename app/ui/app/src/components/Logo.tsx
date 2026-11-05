interface LogoProps {
  size?: number;
  containerClassName?: string;
  showBackground?: boolean;
  showWordmark?: boolean;
}

export default function Logo({
  size = 60,
  containerClassName = "mb-8",
  showWordmark = false,
}: LogoProps = {}) {
  return (
    <div
      className={`flex items-center justify-center gap-3 select-none ${containerClassName}`}
    >
      <img
        src="/ollamapaca.png"
        alt="Ollamapaca"
        width={size}
        height={size}
        className="rounded-2xl select-none"
      />
      {showWordmark && (
        <span className="text-2xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
          Ollamapaca
        </span>
      )}
    </div>
  );
}
