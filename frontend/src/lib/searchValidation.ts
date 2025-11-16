/**
 * Validates a date value in YYYY-MM-DD format.
 * Returns an error message if invalid, or null if valid.
 */
function validateDateValue(dateValue: string, filterType: string): string | null {
    // Check if it looks like a date (should be YYYY-MM-DD)
    const datePattern = /^\d{4}-\d{2}-\d{2}$/
    if (!datePattern.test(dateValue)) {
        return `Invalid date format for ${filterType}:. Expected YYYY-MM-DD (e.g., 2025-01-01)`
    }

    // Parse and validate date components before creating a Date object
    // (JavaScript Date is lenient and will autocorrect invalid dates)
    const parts = dateValue.split('-')
    if (parts.length !== 3) {
        return `Invalid date format for ${filterType}: "${dateValue}"`
    }

    const year = parseInt(parts[0], 10)
    const month = parseInt(parts[1], 10)
    const day = parseInt(parts[2], 10)

    if (isNaN(year) || isNaN(month) || isNaN(day)) {
        return `Invalid date for ${filterType}: "${dateValue}"`
    }

    // Check for impossible dates (e.g., month > 12, day > 31)
    if (month < 1 || month > 12) {
        return `Invalid month in date: ${dateValue}`
    }
    if (day < 1 || day > 31) {
        return `Invalid day in date: ${dateValue}`
    }

    // Validate the date is actually valid (check if Date constructor creates the correct date)
    const date = new Date(year, month - 1, day)
    if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
        return `Invalid date for ${filterType}: "${dateValue}"`
    }

    return null
}

/**
 * Validates date filters (after: and before:) in the query.
 * Returns an error message if invalid, or null if valid.
 */
function validateDateFilters(query: string): string | null {
    const dateFilterPattern = /\b(after|before):(\S+)/gi
    let match
    while ((match = dateFilterPattern.exec(query)) !== null) {
        const filterType = match[1].toLowerCase()
        const dateValue = match[2]
        const error = validateDateValue(dateValue, filterType)
        if (error) {
            return error
        }
    }
    return null
}

/**
 * Validates a search query for common syntax errors.
 * Returns an error message if invalid, or null if valid.
 */
export function validateSearchQuery(query: string): string | null {
    if (!query.trim()) {
        return null // Empty query is allowed
    }

    // Check for empty filter values (e.g., "from:" without value)
    const emptyFilterPattern = /\b(from|to|subject|after|before|folder|label):\s*$/i
    if (emptyFilterPattern.test(query)) {
        return 'Filter value cannot be empty (e.g., "from:" needs a value)'
    }

    // Check for invalid date formats in after: and before: filters
    return validateDateFilters(query)
}
