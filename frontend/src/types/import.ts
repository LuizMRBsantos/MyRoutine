export interface CSVMapping {
  date_column: number
  amount_column: number
  description_column: number
  date_format: string
  /** Extratos costumam trazer despesa como valor negativo. */
  negative_is_expense: boolean
  delimiter: string
  has_header: boolean
}

export interface ParsedEntry {
  occurred_on: string
  amount_cents: number
  kind: 'expense' | 'income'
  raw_description: string
}

export interface ImportPreview {
  mapping: CSVMapping
  /** true quando todas as colunas foram reconhecidas pelo cabeçalho. */
  complete: boolean
  sample: ParsedEntry[]
}

export type ImportEntryStatus = 'pending' | 'imported' | 'ignored' | 'matched'

export interface ImportEntry {
  id: string
  occurred_on: string
  amount_cents: number
  kind: 'expense' | 'income'
  raw_description: string
  suggested_category: string | null
  status: ImportEntryStatus
  /** Presente quando a linha parece duplicata de algo já lançado. */
  matched_transaction_id?: string
  matched_transaction_description?: string
  matched_transaction_date?: string
}

export interface ImportBatch {
  id: string
  filename: string
  credit_card_id: string | null
  account_label: string | null
  created_at: string
  entries?: ImportEntry[]
  new_count: number
  duplicate_count: number
  matched_count: number
  skipped_lines: string[]
}

export interface CategoryRule {
  id: string
  pattern: string
  category: string
  hit_count: number
}
