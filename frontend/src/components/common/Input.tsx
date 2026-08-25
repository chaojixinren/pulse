import { forwardRef, type InputHTMLAttributes } from 'react';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  helperText?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { label, error, helperText, className = '', required, id, ...props },
  ref,
) {
  const hasError = Boolean(error);
  const inputId = id ?? (label ? 'input-' + label : undefined);

  return (
    <div className={'input-wrapper ' + className}>
      {label && (
        <label className="input-label" htmlFor={inputId}>
          {label}
          {required && <span className="input-required">*</span>}
        </label>
      )}
      <div className={'input-container' + (hasError ? ' input-error' : '')}>
        <input
          ref={ref}
          id={inputId}
          className="input-field"
          required={required}
          aria-invalid={hasError}
          {...props}
        />
      </div>
      {hasError && <span className="input-error-text">{error}</span>}
      {!hasError && helperText && <span className="input-helper-text">{helperText}</span>}
    </div>
  );
});
