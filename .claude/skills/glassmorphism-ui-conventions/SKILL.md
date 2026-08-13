---
name: glassmorphism-ui-conventions
description: Define o estilo visual do frontend do MyRoutine (dark mode,
  glassmorphism, design system próprio em React/TypeScript). Use quando
  o usuário pedir para criar ou estilizar componentes de UI, telas novas,
  ou mexer em CSS/estilos do frontend.
---

# Convenções visuais do frontend

## Estilo base
- Dark mode como padrão (não apenas suportado — é o modo primário do
  produto).
- Glassmorphism: superfícies com transparência + blur, não cards sólidos
  opacos. Usar `backdrop-filter: blur(...)` com fundo semi-transparente
  (`rgba`) sobre o background escuro do app.
- Evitar excesso de cor — o produto é uma ferramenta de uso diário
  intenso, não um app consumer buscando engajamento visual. Paleta
  contida, acentos de cor reservados para sinalização real (ex: bloco
  "hábito em risco"), não decoração.

## Regras de aplicação
1. Cor nunca deve ser o único sinal de estado — combinar com ícone ou
   texto (acessibilidade e clareza, especialmente relevante para não
   confundir "queda de consistência" com decoração).
2. Sem vermelho para ausência de dado ou dia não revisado (ver skill
   `habit-review-flow`) — vermelho é reservado para sinais que o usuário
   realmente precisa agir.
3. Componentes de check-in (o fluxo mais usado do app) devem priorizar
   velocidade de interação sobre efeito visual — evitar animações longas
   ou transições que atrasem o toque seguinte.
4. Reaproveitar os tokens de cor e espaçamento já definidos no design
   system do projeto (verificar arquivo de tokens/tema existente antes
   de introduzir novos valores hardcoded).

## Ao criar um componente novo
1. Verificar se já existe um componente similar no design system antes
   de criar um novo do zero.
2. Manter consistência de padding, radius e blur com os componentes
   existentes — não introduzir um novo "estilo de card" sem necessidade.
3. Testar contraste de texto sobre superfícies com blur — glassmorphism
   tende a reduzir legibilidade se não houver texto com peso/cor
   suficiente por cima.
