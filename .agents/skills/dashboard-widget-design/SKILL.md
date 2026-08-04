---
name: dashboard-widget-design
description: Define a hierarquia e os blocos do dashboard de hábitos do
  MyRoutine. Use quando o usuário pedir para criar, redesenhar ou
  adicionar widgets/blocos à tela de hábitos, ao dashboard principal, ou
  pedir novas visualizações e métricas de consistência.
---

# Dashboard de hábitos — hierarquia de blocos

## Regra central
O dashboard não deve virar uma coleção crescente de gráficos
decorativos. Cada bloco existe para responder uma pergunta de decisão
específica. Se um bloco novo não ajuda a decidir algo, ele não deveria
ser permanente na tela — ver princípio 3 da constitution ("o sistema
desaparece quando não agrega valor").

## Blocos existentes, em ordem de prioridade visual
1. **Hoje** — lista operacional dos hábitos do dia, agrupados por
   período. Sempre visível, sempre no topo.
2. **Semana em uma linha** — matriz compacta hábito × 7 dias da semana
   atual. Objetivo: tornar buracos visíveis de forma acionável no curto
   prazo. Não confundir com o heatmap de 90 dias (esse serve para "olhar
   pra trás", a matriz serve para "o que ainda dá pra salvar essa
   semana").
3. **Hábito em risco** — card que só aparece quando há sinal real: taxa
   de conclusão de algum hábito caiu mais que um limiar (ex: >20 pontos
   percentuais) comparando as últimas 2 semanas com as 2 anteriores.
   Ordenado por magnitude da queda. Nunca deve virar um widget fixo
   decorativo — se não há queda, o bloco não aparece.
4. **Heatmap 90 dias** — mantido como está, é o bloco de "olhar pra
   trás com orgulho" / análise de longo prazo.

## O que não incluir neste dashboard
- Rankings, comparação social, badges, pontos (ver product-constitution).
- Notificações ou alertas permanentes sem gatilho de dado real.
- Gráficos "bonitos" sem uma pergunta de decisão associada — se não dá
  pra explicar em uma frase que decisão o gráfico ajuda a tomar, não
  implementar.

## Ao adicionar um bloco novo
Antes de implementar, responder:
1. Que pergunta de decisão esse bloco responde?
2. Ele deveria ser sempre visível ou só aparecer quando há sinal?
   (default: só aparecer quando há sinal, a menos que seja operacional
   como o bloco "Hoje")
3. Ele duplica informação que já existe em outro bloco de forma
   diferente? Se sim, provavelmente não deveria existir.

## Visual
- Ver skill `glassmorphism-ui-conventions` para tokens de cor, dark mode
  e estilo visual do frontend.
