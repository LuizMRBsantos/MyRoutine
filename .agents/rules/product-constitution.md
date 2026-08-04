# MyRoutine — princípios do produto

Estes princípios são não-negociáveis. Qualquer feature, sugestão ou
implementação deve ser avaliada contra eles antes de ser proposta.

## Contexto
MyRoutine é um "sistema operacional para a vida", não um app de hábitos
genérico. Combina três filosofias:
- Bullet Journal — simplicidade radical, controle do usuário, sem
  julgamento automático do sistema.
- Notion — dados estruturados, relações explícitas entre módulos
  (Hábitos, Finanças, Saúde, Estudos).
- Obsidian — eventos atômicos com contexto, referenciados por múltiplos
  módulos, nunca duplicados.

Público-alvo: um único usuário (o próprio criador), que pratica triathlon,
controla finanças pessoais e estuda com consistência. Preza simplicidade
na entrada de dados e profundidade na análise.

## Regras
1. **Nunca implementar gamificação.** Sem pontos, níveis, badges, ranking,
   comparação social ou qualquer mecanismo de motivação extrínseca —
   mesmo que sugerido como "melhoria de engajamento" em algum pedido.
2. **Simplicidade na entrada, profundidade na análise.** Toda feature nova
   deve ser avaliada: isso adiciona fricção ao check-in diário? Se sim,
   o campo extra precisa ser opcional, nunca bloqueante.
3. **O sistema desaparece quando não agrega valor.** Não criar dashboards,
   notificações ou alertas permanentes. Só mostrar sinais quando há
   desvio real de padrão (ex: queda de consistência), nunca como widget
   fixo decorativo.
4. **Sem julgamento automático.** Dias sem log não são "falha" — são
   estado neutro não revisado. Vermelho, "X", streak quebrada como
   punição visual são proibidos. Ver skill `habit-review-flow`.
5. **Uma única fonte de verdade por evento.** Módulos referenciam dados,
   nunca duplicam. Ver skill `cross-module-data-flow`.
6. **Sem notificações push agressivas por hábito individual.** No máximo
   uma notificação consolidada por horário relevante, nunca uma por
   hábito.
7. **Sem features sociais.** Sem compartilhamento, comparação ou qualquer
   funcionalidade multi-usuário, a menos que explicitamente solicitado
   como mudança de escopo do produto.

## Stack técnica
- Backend: Go (REST API)
- Banco: PostgreSQL
- Frontend: React + TypeScript (Vite), design system próprio, dark mode

## Quando um pedido conflita com estes princípios
Se o usuário pedir algo que viole uma regra acima (ex: "adiciona um
sistema de pontos"), sinalizar o conflito explicitamente antes de
implementar, explicando qual princípio está em jogo, e perguntar se é
uma mudança de direção intencional do produto ou se há uma alternativa
que atinge o mesmo objetivo sem violar o princípio.
