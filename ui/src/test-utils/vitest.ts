import { vi, type Mock } from 'vitest'
import type { UseInputValue } from 'react-admin'
import type { FieldError } from 'react-hook-form'

type AnyMock = Mock<(...args: never[]) => unknown>

export const mockFn = <T extends (...args: never[]) => unknown>(
  fn: T,
): AnyMock => vi.mocked(fn) as unknown as AnyMock

export const mockUseInputValue = (
  overrides: Partial<{
    value: unknown
    onChange: (value: unknown) => void
    error: string | undefined
    isTouched: boolean
  }> = {},
): UseInputValue => {
  const onChange = overrides.onChange ?? vi.fn()
  const value = 'value' in overrides ? overrides.value : []
  return {
    id: 'test-input',
    isRequired: false,
    field: {
      name: 'libraryIds',
      value,
      onChange,
      onBlur: vi.fn(),
      ref: vi.fn(),
    },
    formState: {} as UseInputValue['formState'],
    fieldState: {
      error: overrides.error
        ? ({ message: overrides.error, type: 'manual' } as FieldError)
        : undefined,
      invalid: Boolean(overrides.error),
      isDirty: false,
      isTouched: overrides.isTouched ?? false,
      isValidating: false,
    },
  }
}
