---
title: "Vetores de Ataque Analisados"
description: "Listagem detalhada das permissões e configurações auditadas em AWS, Azure e GCP."
weight: 30
---

Este documento detalha os vetores de ataque específicos, permissões críticas e caminhos de escalada de privilégios que o **Squeak the Cloud** monitora e audita em cada provedor de nuvem.

---

## 🟡 AWS (Amazon Web Services)

### 1. Shadow Admin e Escalada de Privilégios
O mecanismo do Squeak mapeia **14 permissões de IAM críticas** que podem ser abusadas por um atacante com acesso inicial limitado para se tornar um Administrador completo (`AdministratorAccess`):

*   `iam:CreateAccessKey`: Permite gerar novas chaves de acesso para outros usuários (incluindo administradores).
*   `iam:CreateLoginProfile` / `iam:UpdateLoginProfile`: Permite definir ou redefinir a senha do console web de outros usuários para acessar via interface gráfica.
*   `iam:AttachUserPolicy` / `iam:AttachRolePolicy` / `iam:AttachGroupPolicy`: Permite vincular políticas privilegiadas (ex: `AdministratorAccess`) ao próprio usuário ou role.
*   `iam:PutUserPolicy` / `iam:PutRolePolicy` / `iam:PutGroupPolicy`: Permite escrever políticas inline com permissões de administrador diretamente na própria identidade.
*   `iam:AddUserToGroup`: Permite adicionar a si mesmo ou a terceiros a grupos de segurança privilegiados (ex: grupo `Admins`).
*   `iam:PassRole`: Permite passar uma role muito privilegiada para um recurso (como uma instância EC2 ou função Lambda) que o atacante controla, executando comandos com essa role.
*   `lambda:CreateFunction` / `lambda:UpdateFunctionCode`: Permite executar código arbitrário (como comandos shell) no ambiente AWS associando roles administrativas à função.
*   `iam:CreatePolicyVersion`: Permite atualizar políticas existentes para versões anteriores ou criar novas versões abusivas ignorando restrições.

### 2. Contas e Credenciais Vulneráveis
*   **Chaves de Acesso Antigas**: Identificação de chaves ativas criadas há mais de 90 dias sem rotação periódica.
*   **Falta de MFA**: Detecção de usuários do console e identidades cruciais sem autenticação de múltiplos fatores (MFA) configurada.

### 3. OPSEC e Evitação de Detecção
*   **Auditoria de CloudTrail**: Varredura das configurações do CloudTrail para verificar em quais regiões os logs de auditoria estão desativados ou não coletam eventos de escrita (`WriteOnly` ou `All`), alertando o operador antes de realizar ações ruidosas.

---

## 🔵 Microsoft Azure

### 1. Sequestro de Assinatura e Escalação de RBAC
Mapeamento de atribuições de roles customizadas e atribuições diretas de permissões administrativas no RBAC do Azure Resource Manager (ARM):

*   `Microsoft.Authorization/roleAssignments/write`: Permite conceder novas permissões e associar funções elevadas a qualquer identidade (incluindo a si próprio).
*   `Microsoft.Authorization/roleDefinitions/write`: Permite modificar a definição de regras de funções existentes ou criar novas funções de controle total.
*   `Microsoft.Compute/virtualMachines/runCommand/action`: Permite executar comandos remotos no sistema operacional de máquinas virtuais do Azure com privilégios de SYSTEM/Root.
*   `Microsoft.Compute/virtualMachines/write`: Permite atualizar configurações da máquina virtual para associar identidades gerenciadas mais fortes (Managed Identities).
*   `Microsoft.Resources/deployments/write`: Permite implantar novos recursos e templates ARM que podem conter configurações inseguras ou credenciais vazadas.
*   `Microsoft.Automation/automationAccounts/runbooks/write`: Permite editar ou criar runbooks administrativos no Azure Automation para executar scripts sob a identidade da conta de automação.
*   `Microsoft.ManagedIdentity/userAssignedIdentities/assign/action`: Permite atribuir identidades gerenciadas por usuários a recursos do Azure que o atacante controla.

### 2. Mapeamento de Identidades Privilegiadas
*   Mapeamento recursivo de usuários, grupos de segurança e **Service Principals** atribuídos a papéis altamente privilegiados no escopo da assinatura (como *Owner*, *Contributor*, *User Access Administrator*).

---

## 🟢 GCP (Google Cloud Platform)

### 1. Escalação Lateral e Impersonificação
Detecção de permissões críticas de IAM em Custom Roles e políticas vinculadas a recursos do projeto:

*   `iam.serviceAccounts.actAs`: Permite agir em nome de uma Service Account (Service Account User), herdando todas as suas permissões.
*   `iam.serviceAccounts.getAccessToken`: Permite gerar tokens de acesso OAuth2 de curta duração para impersonificar a conta de serviço programaticamente.
*   `iam.serviceAccounts.signBlob` / `iam.serviceAccounts.signJwt`: Permite forjar assinaturas e assinar tokens de autenticação como a Service Account.
*   `iam.serviceAccounts.setItemPolicy`: Permite alterar a política de controle de acesso da própria Service Account para conceder acesso a outros operadores.
*   `resourcemanager.projects.setItemPolicy`: Permite reescrever as políticas de IAM de todo o projeto GCP para elevar privilégios.
*   `compute.instances.create`: Permite subir novas instâncias de VM associadas a Service Accounts altamente privilegiadas para abusar de seus escopos e metadados.
*   `deploymentmanager.deployments.create`: Permite criar implantações usando privilégios elevados.

### 2. Chaves de Persistência
*   **User-Managed Keys**: Auditoria de chaves privadas geradas manualmente por usuários (`USER_MANAGED`) para Service Accounts (que não rotacionam automaticamente).
*   **Idade das Chaves**: Detecção de chaves criadas há muito tempo e que podem ter vazado.
*   **Contas Órfãs/Desativadas**: Identificação de Service Accounts desativadas no painel, mas que ainda possuem chaves criptográficas ativas e válidas para autenticação de API.
