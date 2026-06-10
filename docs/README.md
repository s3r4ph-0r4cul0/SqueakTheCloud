# 📖 Documentação do SqueakTheCloud (Hugo + Blowfish)

Este diretório contém o site de documentação oficial do **SqueakTheCloud**, construído utilizando o gerador de sites estáticos **Hugo** e o tema moderno **Blowfish**, totalmente personalizado com uma estética cyberpunk/hacker.

---

## 🛠️ O que foi feito (Customizações)

Fizemos uma reestruturação completa da documentação para torná-la profissional, moderna e alinhada à identidade visual do projeto:

1. **Adoção do Tema Blowfish**: Migramos a documentação para o Hugo estruturando as seções no tema Blowfish.
2. **Estética Cyberpunk ("Hacker Yellow")**: Criamos um esquema de cores personalizado (`hacker-yellow.css`) baseado em tons escuros profundos (charcoal/off-black) e destaques em amarelo neon (`#f1c40f`).
3. **Bento Grid na Homepage**: Substituímos as listas simples do arquivo inicial por um Grid Bento assimétrico e dinâmico no layout principal. Cada bloco do grid possui:
   - Micro-interações de movimento e escala ao passar o mouse.
   - Efeito cascata (*staggered reveal*) suave ao carregar a página.
   - Efeitos magnéticos sutis e animação nas setas indicadoras.
4. **Efeito Liquid Glass**: Implementamos recipientes com efeito de vidro fosco translúcido (*glassmorphism*) refinado com bordas internas simulando refração de luz.
5. **Cabeçalho e Rodapé Globais Edge-to-Edge**: Redesenhamos o header e o footer para que ocupassem toda a largura da tela de forma fluida, aplicando desfoque de fundo e bordas de neon amarelo, mantendo o conteúdo interno alinhado ao centro.
6. **Correção de Renderização do Mermaid**: Ajustamos os diagramas de arquitetura para usar o shortcode do Hugo `{{< mermaid >}}`, corrigindo falhas onde o código era exibido cru.
7. **Resolução de Bugs de Sombras**: Corrigimos bugs visuais onde o brilho das letras no cabeçalho ficava quadrado (causado pelo clipping de `overflow: hidden` do Tailwind e conflito de renderização do `backdrop-filter`).

---

## 📌 Pré-requisitos e Dependências

Para rodar e compilar a documentação localmente, você precisa de:

1. **Hugo Extended**: A versão "Extended" do Hugo é obrigatória porque o tema Blowfish utiliza compilação de arquivos Sass/SCSS nativos.
   * *O executável do Hugo Extended já está pré-instalado na raiz do projeto dentro da pasta `bin/`.*
2. **Blowfish Theme**: O tema está localizado em `themes/blowfish/`.

---

## 🚀 Como Executar Localmente

Você pode iniciar o servidor de desenvolvimento do Hugo de duas formas:

### Opção 1: Usando o binário local do projeto (Recomendado)
Acesse a pasta `docs/` e execute o binário do Hugo que está na pasta `bin/` na raiz do projeto:
```bash
cd docs
../bin/hugo server
```

### Opção 2: Usando o Hugo instalado globalmente no seu sistema (caso tenha instalado)
```bash
cd docs
hugo server
```

O servidor iniciará localmente e a documentação estará disponível em:
👉 **[http://localhost:1313/](http://localhost:1313/)**

Qualquer alteração feita nos arquivos Markdown ou CSS será atualizada automaticamente no navegador (Live Reload).

---

## 📂 Estrutura de Pastas da Documentação

* `config/_default/`: Configurações gerais do site, menus, cabeçalho, rodapé e parâmetros do tema Blowfish.
* `content/`: Arquivos de conteúdo em Markdown.
  * `content/_index.md`: Conteúdo textual da página inicial (renderizado dentro do Bento Grid customizado).
  * `content/docs/`: Artigos principais de documentação (Instalação, Como Usar, Vetores de Ataque, etc.).
* `layouts/partials/`: Layouts customizados em HTML.
  * `layouts/partials/home/custom.html`: Onde o Grid Bento e a estrutura personalizada da página inicial estão codificados.
  * `layouts/partials/header/basic.html`: Cabeçalho customizado.
  * `layouts/partials/footer.html`: Rodapé customizado.
* `assets/css/schemes/hacker-yellow.css`: Arquivo CSS onde residem todas as variáveis de cores, efeitos de brilho, animações dos cartões e o visual Liquid Glass.

---

## 📝 Como Adicionar Nova Documentação

Para expandir ou adicionar novas páginas ao site de documentação, siga as instruções e a lógica abaixo:

### 1. Criar o Arquivo Markdown
Todos os artigos principais de documentação devem ser criados na pasta `docs/content/docs/` com a extensão `.md`.
* Exemplo: `docs/content/docs/meu-novo-guia.md`

### 2. Definir o Front Matter (Metadados do Topo)
Toda nova página precisa ter um bloco de metadados delimitado por `---` logo no início do arquivo. Use a seguinte estrutura padrão:
```markdown
---
title: "Título que Aparecerá na Página"
description: "Uma descrição curta para SEO e pré-visualização"
lead: "Texto curto em destaque que inicia o artigo (opcional)"
date: 2026-06-05
draft: false
weight: 40
---
```

### 3. Lógica de Ordenação e Estrutura
* **Ordem no Menu Lateral**: É controlada pela propriedade `weight` no Front Matter.
  * `weight: 10` (Aparece no topo do menu lateral - ex: Introdução/Instalação)
  * `weight: 20` (Aparece abaixo - ex: Como Usar)
  * Quanto maior o número, mais para baixo a página ficará listada.
* **Menus Aninhados**: Se quiser criar sub-páginas, crie uma subpasta dentro de `docs/content/docs/` (ex: `docs/content/docs/aws/`), adicione um arquivo `_index.md` nela com metadados do menu pai, e coloque as sub-páginas dentro.

### 4. Boas Práticas do Tema
* **Diagramas**: Sempre use o shortcode `{{< mermaid >}}` para diagramas de fluxo em vez de blocos genéricos de código ` ```mermaid `.
  ```markdown
  {{< mermaid >}}
  graph TD
      A --> B
  {{< /mermaid >}}
  ```
* **Links Internos**: Para criar links seguros entre páginas da documentação sem quebrar as URLs, use o sistema de referência do Hugo:
  `[Texto do Link]({{< ref "getting-started.md" >}})`

