import * as z from 'zod'
export const createFormSchema = (t) => z.object({
    id: z.number().optional(),
    applies_to: z.enum(['contact', 'conversation', 'organization'], {
        required_error: t('globals.messages.required'),
    }),
    name: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .min(3, {
            message: t('validation.minmax', {
                min: 3,
                max: 140,
            })
        })
        .max(140, {
            message: t('validation.minmax', {
                min: 3,
                max: 140,
            })
        }),
    key: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .min(3, {
            message: t('validation.minmax', {
                min: 3,
                max: 140,
            })
        })
        .max(140, {
            message: t('validation.minmax', {
                min: 3,
                max: 140,
            })
        })
        .regex(/^[a-z0-9_]+$/, {
            message: t('validation.invalidKey'),
        }),
    description: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .min(3, {
            message: t('validation.minmax', {
                min: 3,
                max: 300,
            })
        })
        .max(300, {
            message: t('validation.minmax', {
                min: 3,
                max: 300,
            })
        }),
    data_type: z.enum(['text', 'number', 'checkbox', 'date', 'link', 'list', 'multi_select'], {
        required_error: t('globals.messages.required'),
    }),
    regex: z.string().optional(),
    regex_hint: z.string().optional(),
    values: z.array(z.string())
        .default([])
})
    .superRefine((data, ctx) => {
        if (data.data_type === 'multi_select' && data.applies_to !== 'conversation') {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: t('admin.customAttributes.multiSelectConversationOnly'),
                path: ['data_type'],
            });
        }
        if (data.data_type === 'list' || data.data_type === 'multi_select') {
            // List and multi-select types need at least one value.
            if (!data.values || data.values.length === 0) {
                ctx.addIssue({
                    code: z.ZodIssueCode.too_small,
                    minimum: 1,
                    type: "array",
                    inclusive: true,
                    message: t('globals.messages.required'),
                    path: ['values'],
                });
            }
        }
    });
