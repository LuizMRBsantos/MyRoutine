export interface ParsedLine {
  originalText: string;
  bulletType: 'task_pending' | 'task_done' | 'task_migrated' | 'note' | 'event' | 'none';
  transaction?: {
    value: number;
    category: string;
    method?: string;
  };
  health?: {
    type: string;
    distance?: number;
    time?: number;
    rpe?: number;
    macros?: string;
  };
  links: string[];
}

export function parseLine(line: string): ParsedLine {
  let bulletType: ParsedLine['bulletType'] = 'none';
  const trimmed = line.trim();

  if (trimmed.startsWith('•')) bulletType = 'task_pending';
  else if (/^[xX]\s/.test(trimmed)) bulletType = 'task_done';
  else if (trimmed.startsWith('>')) bulletType = 'task_migrated';
  else if (trimmed.startsWith('-')) bulletType = 'note';
  else if (trimmed.startsWith('○')) bulletType = 'event';

  const linksMatch = trimmed.match(/\[\[(.*?)\]\]/g);
  const links = linksMatch ? linksMatch.map(l => l.replace('[[', '').replace(']]', '')) : [];

  let transaction;
  // Match "$ 50 Almoço #alimentação" (Note: \S+ is used instead of \w+ to allow accented chars)
  const financeMatch = trimmed.match(/^\$\s*(\d+(?:[.,]\d{1,2})?)\s+(.*?)(?:\s+#(\S+))?\s*$/i);
  if (financeMatch) {
    transaction = {
      value: parseFloat(financeMatch[1].replace(',', '.')),
      category: financeMatch[3] ? financeMatch[3].toLowerCase() : 'geral',
      // Note: The description part (Almoço) could be used as well if needed.
    };
  }

  let health;
  // Basic matching for health records (Running, Biking, Swimming)
  const isRun = trimmed.includes('🏃') || /corrida/i.test(trimmed);
  const isBike = trimmed.includes('🚴') || /ciclismo/i.test(trimmed);
  const isSwim = trimmed.includes('🏊') || /natação/i.test(trimmed);
  
  if (isRun || isBike || isSwim) {
    const distMatch = trimmed.match(/(\d+(?:[.,]\d+)?)\s*(km|m)/i);
    const rpeMatch = trimmed.match(/RPE\s*(\d+)/i);
    const timeMatch = trimmed.match(/(\d+)\s*(min|h|horas)/i);

    health = {
      type: isRun ? 'run' : (isBike ? 'bike' : 'swim'),
      distance: distMatch ? parseFloat(distMatch[1].replace(',', '.')) : undefined,
      rpe: rpeMatch ? parseInt(rpeMatch[1], 10) : undefined,
      time: timeMatch ? parseInt(timeMatch[1], 10) : undefined, // simplified
    };
  }

  return {
    originalText: line,
    bulletType,
    transaction,
    health,
    links
  };
}

export function parseNoteContent(content: string): ParsedLine[] {
  return content.split('\n').map(parseLine);
}
