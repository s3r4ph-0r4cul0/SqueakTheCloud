---
title: "Uso da CLI"
description: "Opções de comandos, parâmetros e exemplos práticos de execução do Squeak the Cloud."
weight: 20
---

Este guia demonstra como executar o **Squeak the Cloud** a partir do terminal, suas principais opções de ajuda e exemplos práticos para rodar análises em diferentes nuvens.

---

## 🧭 Diagrama de Funcionamento

O mecanismo interno do **Squeak the Cloud** carrega as credenciais e direciona a execução de acordo com o provedor passado como argumento CLI. Os resultados parciais são gerados de forma assíncrona/modular e, ao fim, consolidados em um relatório central:

{{< mermaid >}}
graph TD
    A[Credenciais do Operador / Env Vars] --> B(Squeak the Cloud Engine)
    B --> C{Provedor Selecionado}
    C -->|--provider aws| D[AWS IAM & CloudTrail API]
    C -->|--provider azure| E[Azure Resource Manager RBAC]
    C -->|--provider gcp| F[GCP Resource Manager & IAM]
    D --> G[Análise Estática de Políticas & Shadow Admin]
    E --> H[Resolução de Role Assignments & Privilégios]
    F --> I[Enumeração de Service Accounts & Custom Roles]
    G --> J[results/aws_*.json]
    H --> K[results/azure_*.json]
    I --> L[results/gcp_*.json]
    J --> M(results/consolidated_results.json)
    K --> M
    L --> M
{{< /mermaid >}}

---

## 💻 Ajuda da CLI

Para listar todos os parâmetros e opções disponíveis diretamente na ferramenta, execute:

```bash
./squeak-audit -h
```

### Parâmetros Suportados

| Flag | Tipo | Descrição | Obrigatório? |
| :--- | :--- | :--- | :--- |
| `-provider` | `string` | Define o provedor de nuvem para auditoria. Valores suportados: `aws`, `azure`, `gcp`. | **Sim** |

---

## 🚀 Exemplos Práticos de Execução

### 🟡 Auditoria AWS (Amazon Web Services)
Gera o relatório de segurança do IAM e de auditoria de OPSEC com foco no CloudTrail e privilégios críticos:
```bash
./squeak-audit --provider aws
```

### 🔵 Auditoria Microsoft Azure
Mapeia o RBAC (Role-Based Access Control) na assinatura atual para encontrar atribuições perigosas e privilégios ocultos:
```bash
# Definindo a subscription alvo (se necessário)
export AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"

# Executando a auditoria
./squeak-audit --provider azure
```

### 🟢 Auditoria GCP (Google Cloud Platform)
Enumera Custom Roles e chaves persistentes de Service Accounts no projeto ativo:
```bash
# Definindo o projeto alvo (se necessário)
export GCP_PROJECT_ID="target-project-id"

# Executando a auditoria
./squeak-audit --provider gcp
```
