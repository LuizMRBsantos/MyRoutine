import { describe, it, expect } from 'vitest';
import { parseLine, parseNoteContent } from './index';

// O parser é a porta de entrada de tudo que o diário envia ao servidor:
// um erro aqui vira transação com valor errado ou treino no hábito errado.

describe('bullets do Bullet Journal', () => {
  it('reconhece cada tipo de bullet', () => {
    expect(parseLine('• Comprar pão').bulletType).toBe('task_pending');
    expect(parseLine('x Terminei o relatório').bulletType).toBe('task_done');
    expect(parseLine('> Adiado para amanhã').bulletType).toBe('task_migrated');
    expect(parseLine('- Anotação solta').bulletType).toBe('note');
    expect(parseLine('○ Aniversário da Ana').bulletType).toBe('event');
  });

  it('aceita X maiúsculo para tarefa concluída', () => {
    expect(parseLine('X Terminei').bulletType).toBe('task_done');
  });

  it('ignora indentação', () => {
    expect(parseLine('    • Subtarefa').bulletType).toBe('task_pending');
  });

  it('trata linha sem marcador como texto livre', () => {
    expect(parseLine('Só um pensamento').bulletType).toBe('none');
    expect(parseLine('').bulletType).toBe('none');
  });

  // "x" sozinho no início de palavra não deve virar tarefa concluída
  it('não confunde palavra iniciada por x com tarefa concluída', () => {
    expect(parseLine('xícara de café').bulletType).toBe('none');
  });
});

describe('wiki-links', () => {
  it('extrai um link', () => {
    expect(parseLine('Falei com [[João]] hoje').links).toEqual(['João']);
  });

  it('extrai vários links da mesma linha', () => {
    expect(parseLine('[[Projeto Alpha]] e [[Projeto Beta]]').links)
      .toEqual(['Projeto Alpha', 'Projeto Beta']);
  });

  it('devolve lista vazia quando não há links', () => {
    expect(parseLine('sem links').links).toEqual([]);
  });
});

describe('transações', () => {
  it('extrai valor, descrição e categoria', () => {
    const { transaction } = parseLine('$ 50 Almoço #alimentacao');
    expect(transaction).toEqual({
      value: 50,
      description: 'Almoço',
      category: 'alimentacao',
    });
  });

  it('aceita centavos com vírgula ou ponto', () => {
    expect(parseLine('$ 25,90 Café').transaction?.value).toBe(25.9);
    expect(parseLine('$ 25.90 Café').transaction?.value).toBe(25.9);
  });

  it('aceita valor colado no cifrão', () => {
    expect(parseLine('$45 Uber').transaction?.value).toBe(45);
  });

  it('usa "other" quando não há hashtag de categoria', () => {
    // 'other' é a categoria default do backend — precisa bater
    expect(parseLine('$ 30 Livro').transaction?.category).toBe('other');
  });

  it('normaliza a categoria para minúsculas', () => {
    expect(parseLine('$ 30 Livro #Educacao').transaction?.category).toBe('educacao');
  });

  it('aceita categoria acentuada', () => {
    expect(parseLine('$ 50 Almoço #alimentação').transaction?.category).toBe('alimentação');
  });

  it('não extrai transação de linha sem cifrão', () => {
    expect(parseLine('gastei 50 reais no almoço').transaction).toBeUndefined();
  });
});

describe('registros de saúde', () => {
  it('extrai corrida com distância e RPE', () => {
    const { health } = parseLine('🏃 Corrida 8km RPE 6');
    expect(health?.type).toBe('run');
    expect(health?.distance).toBe(8);
    expect(health?.rpe).toBe(6);
  });

  it('reconhece a modalidade por palavra ou por emoji', () => {
    expect(parseLine('corrida leve').health?.type).toBe('run');
    expect(parseLine('🏃 30min').health?.type).toBe('run');
    expect(parseLine('ciclismo 40km').health?.type).toBe('bike');
    expect(parseLine('🚴 40km').health?.type).toBe('bike');
    expect(parseLine('natação 2km').health?.type).toBe('swim');
    expect(parseLine('🏊 2km').health?.type).toBe('swim');
  });

  it('aceita distância decimal', () => {
    expect(parseLine('corrida 8,2km').health?.distance).toBe(8.2);
    expect(parseLine('corrida 8.2km').health?.distance).toBe(8.2);
  });

  it('extrai duração em minutos', () => {
    expect(parseLine('corrida 45min').health?.time).toBe(45);
  });

  it('deixa campos ausentes como undefined em vez de zero', () => {
    const { health } = parseLine('corrida tranquila');
    expect(health?.type).toBe('run');
    expect(health?.distance).toBeUndefined();
    expect(health?.rpe).toBeUndefined();
  });

  it('não extrai saúde de linha sem modalidade', () => {
    expect(parseLine('caminhei um pouco').health).toBeUndefined();
  });
});

describe('parseNoteContent', () => {
  it('processa um diário completo linha a linha', () => {
    const content = [
      '• Estudar OAC',
      'x Treino da manhã',
      '$ 45 Almoço #alimentacao',
      '🏃 Corrida 8km RPE 6',
      '- Reunião com [[Orientador]]',
    ].join('\n');

    const lines = parseNoteContent(content);
    expect(lines).toHaveLength(5);

    const transactions = lines.filter((l) => l.transaction);
    const workouts = lines.filter((l) => l.health);
    const links = lines.flatMap((l) => l.links);

    expect(transactions).toHaveLength(1);
    expect(transactions[0].transaction?.value).toBe(45);
    expect(workouts).toHaveLength(1);
    expect(workouts[0].health?.distance).toBe(8);
    expect(links).toEqual(['Orientador']);
  });

  it('preserva linhas vazias para não bagunçar a numeração', () => {
    expect(parseNoteContent('a\n\nb')).toHaveLength(3);
  });

  it('devolve uma linha para conteúdo vazio', () => {
    expect(parseNoteContent('')).toHaveLength(1);
  });

  it('mantém o texto original de cada linha', () => {
    expect(parseNoteContent('• Tarefa')[0].originalText).toBe('• Tarefa');
  });
});
