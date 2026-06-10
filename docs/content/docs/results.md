---
title: "Estrutura de Resultados"
description: "Formato dos relatórios JSON e do índice consolidated_results.json gerados pelo Squeak the Cloud."
weight: 40
---

Este documento explica como o **Squeak the Cloud** organiza seus relatórios de auditoria e descreve a estrutura dos arquivos JSON gerados ao final de cada execução.

---

## 📁 Diretório de Resultados

Todas as informações coletadas são gravadas de forma detalhada e modular no diretório `./results/` na raiz do projeto onde a ferramenta foi executada:

```text
SqueakTheCloud/
└── results/
    ├── aws_account_security.json
    ├── aws_iam_role_admin-role.json
    ├── aws_logging_security.json
    ├── azure_role_assignments.json
    ├── gcp_service_accounts.json
    └── consolidated_results.json
```

---

## 🔍 Nomenclatura dos Relatórios por Provedor

Dependendo do provedor selecionado, diferentes relatórios granulares serão criados:

### 🟡 AWS (Amazon Web Services)
*   `aws_account_security.json`: Status geral da conta (idade das chaves dos usuários, configurações de MFA, chaves ativas).
*   `aws_logging_security.json`: Status do CloudTrail, status de logging por região, status de criptografia de logs (KMS).
*   `aws_iam_role_*.json`: Detalhes específicos de políticas de roles identificadas com permissões de escalada.
*   `aws_iam_user_*.json`: Detalhes de usuários com permissões de escalada ou shadow admin.
*   `aws_oidc_provider_*.json` / `aws_saml_provider_*.json`: Provedores de identidade confiáveis e suas configurações.

### 🔵 Microsoft Azure
*   `azure_role_*.json`: Detalhes das definições de funções customizadas encontradas.
*   `azure_role_assignment_*.json`: Associação de papéis, identidades atribuídas, escopos e status de permissão de escrita de RBAC.

### 🟢 GCP (Google Cloud Platform)
*   `gcp_role_*.json`: Definições de Custom Roles associadas a permissões críticas.
*   `gcp_service_account_*.json`: Lista de Service Accounts enumeradas, metadados de chaves gerenciadas por usuários (`USER_MANAGED`) e status de ativação da conta.

---

## 📊 Índice Consolidado (`consolidated_results.json`)

Para facilitar a integração com pipelines de CI/CD ou importadores automáticos de dados (ex: scripts Python que importam para o BloodHound), a ferramenta cria ou atualiza um arquivo central de resumo chamado `consolidated_results.json` na raiz da pasta `results/`.

Este arquivo indica qual provedor foi executado por último e lista todos os relatórios gerados que contêm as descobertas específicas:

```json
{
  "provider": "aws",
  "files": [
    "aws_account_security.json",
    "aws_iam_role_admin-role.json",
    "aws_iam_user_operator-user.json",
    "aws_logging_security.json"
  ]
}
```

Dessa forma, ferramentas externas precisam apenas ler o arquivo consolidated para saber quais relatórios de auditoria ler e analisar.
