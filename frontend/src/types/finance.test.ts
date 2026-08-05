import { describe, it, expect } from 'vitest'
import { parseAmountToCents, formatCents, categoryLabel, categoryIcon } from './finance'

// O backend guarda dinheiro em centavos inteiros; toda a conversão acontece
// aqui. Um erro nesta função vira erro de valor no extrato do usuário.
describe('parseAmountToCents', () => {
  it('converte decimal com vírgula (padrão pt-BR)', () => {
    expect(parseAmountToCents('25,90')).toBe(2590)
    expect(parseAmountToCents('0,99')).toBe(99)
  })

  it('converte decimal com ponto', () => {
    expect(parseAmountToCents('25.90')).toBe(2590)
  })

  it('converte inteiro sem separador decimal', () => {
    expect(parseAmountToCents('45')).toBe(4500)
  })

  it('trata ponto como separador de milhar', () => {
    expect(parseAmountToCents('1.234,56')).toBe(123456)
    expect(parseAmountToCents('1.000')).toBe(100000)
  })

  it('ignora espaços em volta', () => {
    expect(parseAmountToCents('  25,90  ')).toBe(2590)
  })

  it('arredonda para o centavo mais próximo', () => {
    expect(parseAmountToCents('10,999')).toBe(1100)
  })

  it('rejeita entrada inválida, vazia ou não positiva', () => {
    expect(parseAmountToCents('')).toBeNull()
    expect(parseAmountToCents('abc')).toBeNull()
    expect(parseAmountToCents('0')).toBeNull()
    expect(parseAmountToCents('-10')).toBeNull()
  })
})

describe('formatCents', () => {
  it('formata centavos como moeda brasileira', () => {
    //   é o espaço não separável que o Intl insere após "R$"
    expect(formatCents(2590).replace(/ /g, ' ')).toBe('R$ 25,90')
    expect(formatCents(0).replace(/ /g, ' ')).toBe('R$ 0,00')
    expect(formatCents(123456).replace(/ /g, ' ')).toBe('R$ 1.234,56')
  })

  it('formata valores negativos (saldo no vermelho)', () => {
    expect(formatCents(-5000)).toContain('25,00'.slice(0, 0) + '50,00')
  })

  it('faz round-trip com parseAmountToCents', () => {
    for (const input of ['25,90', '1.234,56', '0,01', '999']) {
      const cents = parseAmountToCents(input)!
      const formatted = formatCents(cents).replace(/[R$ \s.]/g, '').replace(',', '.')
      expect(Math.round(parseFloat(formatted) * 100)).toBe(cents)
    }
  })
})

describe('categorias', () => {
  it('traduz categorias conhecidas', () => {
    expect(categoryLabel('alimentacao')).toBe('Alimentação')
    expect(categoryIcon('transporte')).toBe('🚗')
  })

  it('devolve a própria chave quando a categoria é desconhecida', () => {
    // O parser do mobile pode criar categorias novas a partir de #hashtags
    expect(categoryLabel('viagem')).toBe('viagem')
    expect(categoryIcon('viagem')).toBe('📦')
  })
})
