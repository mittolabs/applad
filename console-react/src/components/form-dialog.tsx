import { type ReactNode } from 'react';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import { Button } from './ui/button';
import { Label } from './ui/label';
import { Input } from './ui/input';
import { Textarea } from './ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';
import { cn } from '@/lib/utils';

/* Ports app_dialog.dart — a controlled dialog shell with header/body/footer,
 * plus FormField / SelectField and a destructive/loading action button.
 * Use <FormDialog> for create/edit forms and <ConfirmDialog> for confirms. */

export function FormDialog({
  open,
  onOpenChange,
  title,
  subtitle,
  children,
  submitLabel = 'Save',
  cancelLabel = 'Cancel',
  onSubmit,
  loading,
  destructive,
  submitDisabled,
  width = 440,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  subtitle?: string;
  children: ReactNode;
  submitLabel?: string;
  cancelLabel?: string;
  onSubmit?: () => void;
  loading?: boolean;
  destructive?: boolean;
  submitDisabled?: boolean;
  width?: number;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent width={width}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {subtitle && <DialogDescription>{subtitle}</DialogDescription>}
        </DialogHeader>
        <DialogBody>
          <form
            id="form-dialog"
            onSubmit={(e) => {
              e.preventDefault();
              onSubmit?.();
            }}
            className="flex flex-col gap-4"
          >
            {children}
          </form>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)} type="button">
            {cancelLabel}
          </Button>
          {onSubmit && (
            <Button
              type="submit"
              form="form-dialog"
              variant={destructive ? 'destructive' : 'primary'}
              loading={loading}
              disabled={submitDisabled}
            >
              {submitLabel}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/* Confirm dialog — ports the destructive showAppDialog confirm pattern used by
 * DataTable row-delete and account/project deletes. */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  message,
  confirmLabel = 'Delete',
  cancelLabel = 'Cancel',
  onConfirm,
  loading,
  destructive = true,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  onConfirm: () => void;
  loading?: boolean;
  destructive?: boolean;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent width={400}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <div className="text-[length:var(--text-body)] text-text-secondary">
            {message}
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {cancelLabel}
          </Button>
          <Button
            variant={destructive ? 'destructive' : 'primary'}
            loading={loading}
            onClick={onConfirm}
          >
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/* Labeled form field — ports app_dialog.dart AppDialogField. */
export function FormField({
  label,
  hint,
  error,
  children,
  className,
}: {
  label?: string;
  hint?: string;
  error?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      {label && <Label>{label}</Label>}
      {children}
      {error ? (
        <span className="text-[length:var(--text-caption)] text-[var(--color-danger)]">
          {error}
        </span>
      ) : (
        hint && (
          <span className="text-[length:var(--text-caption)] text-text-muted">{hint}</span>
        )
      )}
    </div>
  );
}

/* Text field — label + Input. */
export function TextField({
  label,
  hint,
  error,
  ...props
}: {
  label?: string;
  hint?: string;
  error?: string;
} & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <FormField label={label} hint={hint} error={error}>
      <Input {...props} />
    </FormField>
  );
}

/* Multiline field — label + Textarea. */
export function TextAreaField({
  label,
  hint,
  error,
  ...props
}: {
  label?: string;
  hint?: string;
  error?: string;
} & React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <FormField label={label} hint={hint} error={error}>
      <Textarea {...props} />
    </FormField>
  );
}

/* Labeled select — ports app_dialog.dart AppSelectField. */
export function SelectField<T extends string>({
  label,
  hint,
  value,
  onChange,
  options,
  placeholder = 'Select…',
}: {
  label?: string;
  hint?: string;
  value: T | undefined;
  onChange: (value: T) => void;
  options: { value: T; label: string }[];
  placeholder?: string;
}) {
  return (
    <FormField label={label} hint={hint}>
      <Select value={value} onValueChange={(v) => onChange(v as T)}>
        <SelectTrigger>
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {options.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </FormField>
  );
}
