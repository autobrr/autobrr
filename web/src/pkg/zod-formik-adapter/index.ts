import { core, z } from "zod/v4";

export class ValidationError extends Error {
    public name = "ValidationError";

    public inner: Array<{ path: string; message: string }> = [];

    public constructor(message: string) {
        super(message);
    }
}

function createValidationError(e: z.ZodError) {
    const error = new ValidationError(e.message);
    error.inner = e.issues.map((err) => ({
        message: err.message,
        path: err.path.join("."),
    }));

    return error;
}

/**
 * Wrap your zod schema in this function when providing it to Formik's validation schema prop
 * @param schema The zod schema
 * @param params
 * @returns An object containing the `validate` method expected by Formik
 */
export function toFormikValidationSchema<T>(
    schema: z.ZodSchema<T>,
    params?: Partial<core.ParseContext<core.$ZodIssue>>
): { validate: (obj: T) => Promise<void> } {
    return {
        async validate(obj: T) {
            try {
                await schema.parseAsync(obj, params);
            } catch (err: unknown) {
                throw createValidationError(err as z.ZodError<T>);
            }
        },
    };
}

function createValidationResult(error: z.ZodError) {
    const result: Record<string, string> = {};

    for (const x of error.issues) {
        result[x.path.filter(Boolean).join(".")] = x.message;
    }

    return result;
}

/**
 * Wrap your zod schema in this function when providing it to Formik's validate prop
 * @param schema The zod schema
 * @param params
 * @returns An validate function as expected by Formik
 */
export function toFormikValidate<T>(
    schema: z.ZodSchema<T>,
    params?: Partial<core.ParseContext<core.$ZodIssue>>
) {
    return async (values: T) => {
        const result = await schema.safeParseAsync(values, params);
        if (!result.success) {
            return createValidationResult(result.error);
        }
    };
}