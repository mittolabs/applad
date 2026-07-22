import { SignalHigh, Wifi, BatteryFull } from 'lucide-react';

/* Ports _SmsPhonePreview / _PushPhonePreview from messaging_page.dart —
 * a CSS phone frame that updates live as the user types. */

const FRAME =
  'flex flex-col overflow-hidden rounded-[26px] border-2 bg-[#111113]';
const FRAME_BORDER = 'rgba(255,255,255,0.15)';

function StatusBar() {
  return (
    <div className="flex h-7 items-center gap-0.5 px-3.5">
      <span className="text-[10px] font-semibold text-white/80">9:41</span>
      <span className="flex-1" />
      <SignalHigh size={10} className="text-white/70" />
      <Wifi size={10} className="text-white/70" />
      <BatteryFull size={10} className="text-white/70" />
    </div>
  );
}

const EMPTY_HINT =
  'Enter your message in the input field on the left to see it here';

export function SmsPhonePreview({ message }: { message: string }) {
  return (
    <div
      className={FRAME}
      style={{ width: 155, height: 290, borderColor: FRAME_BORDER }}
    >
      <StatusBar />
      <div className="flex flex-1 flex-col items-center pt-2">
        <div className="h-9 w-9 rounded-full bg-white/10" />
        <div className="mt-1 text-[9px] text-white/50">Today 4:37 PM</div>
        {message ? (
          <div className="mt-2 flex w-full justify-end px-2.5">
            <div className="rounded-[12px] bg-[#2196F3] p-2 text-[9px] text-white">
              {message}
            </div>
          </div>
        ) : (
          <div className="mt-2 px-4 text-center text-[9px] text-white/30">
            {EMPTY_HINT}
          </div>
        )}
      </div>
    </div>
  );
}

export function PushPhonePreview({
  title,
  message,
}: {
  title: string;
  message: string;
}) {
  const hasContent = title.length > 0 || message.length > 0;
  return (
    <div
      className={FRAME}
      style={{ width: 155, height: 320, borderColor: FRAME_BORDER }}
    >
      <StatusBar />
      <div className="flex flex-1 items-center justify-center">
        {hasContent ? (
          <div className="mx-2 rounded-[12px] bg-white/10 p-2.5">
            <div className="flex items-center gap-1">
              <span className="h-3.5 w-3.5 rounded-full bg-[var(--color-accent)]" />
              <span className="text-[9px] text-white/50">App</span>
              <span className="flex-1" />
              <span className="text-[9px] text-white/40">now</span>
            </div>
            {title && (
              <div className="mt-1 text-[10px] font-semibold text-white">
                {title}
              </div>
            )}
            {message && (
              <div className="mt-0.5 line-clamp-3 text-[9px] text-white/70">
                {message}
              </div>
            )}
          </div>
        ) : (
          <div className="px-3.5 text-center text-[9px] text-white/30">
            {EMPTY_HINT}
          </div>
        )}
      </div>
    </div>
  );
}
