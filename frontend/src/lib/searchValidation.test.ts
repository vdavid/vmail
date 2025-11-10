import { describe, expect, it } from 'vitest'

import { validateSearchQuery } from './searchValidation'

describe('validateSearchQuery', () => {
    it('returns null for empty query', () => {
        expect(validateSearchQuery('')).toBeNull()
        expect(validateSearchQuery('   ')).toBeNull()
    })

    it('returns null for valid plain text query', () => {
        expect(validateSearchQuery('test query')).toBeNull()
    })

    it('returns error for empty from: value', () => {
        const error = validateSearchQuery('from:')
        expect(error).toContain('empty')
    })

    it('returns error for empty to: value', () => {
        const error = validateSearchQuery('to:')
        expect(error).toContain('empty')
    })

    it('returns error for invalid date format in after:', () => {
        const error = validateSearchQuery('after:invalid-date')
        expect(error).toContain('Invalid date format')
    })

    it('returns error for invalid date format in before:', () => {
        const error = validateSearchQuery('before:not-a-date')
        expect(error).toContain('Invalid date format')
    })

    it('returns error for invalid month in date', () => {
        const error = validateSearchQuery('after:2025-13-01')
        expect(error).toContain('Invalid month')
    })

    it('returns error for invalid day in date', () => {
        const error = validateSearchQuery('after:2025-01-32')
        expect(error).toContain('Invalid day')
    })

    it('returns null for valid date format', () => {
        expect(validateSearchQuery('after:2025-01-01')).toBeNull()
        expect(validateSearchQuery('before:2025-12-31')).toBeNull()
    })

    it('returns null for valid filter queries', () => {
        expect(validateSearchQuery('from:george')).toBeNull()
        expect(validateSearchQuery('to:alice')).toBeNull()
        expect(validateSearchQuery('subject:meeting')).toBeNull()
        expect(validateSearchQuery('folder:Inbox')).toBeNull()
    })

    it('returns null for complex valid queries', () => {
        expect(validateSearchQuery('from:george after:2025-01-01 cabbage')).toBeNull()
    })
})
