# 🐁 Squeak the Cloud 🐁 - Auditor de Postura de Segurança Multi-Cloud (Go)

O **SqueakTheCloud** evoluiu de um script Bash de extração de metadados para uma ferramenta avançada e abrangente de auditoria de postura de segurança e análise ativa de identidades/permissões (**CSPM & CIEM**), escrita em **Go**.

A ferramenta agora suporta auditorias detalhadas na **AWS**, **Azure** e **GCP**, com foco especial em detectar caminhos ocultos de escalada de privilégios (**Shadow Admin**) e falhas de configuração de postura de identidades.

<p align="center">
  <img src="https://media3.giphy.com/media/v1.Y2lkPTc5MGI3NjExMHJmbnI4MHBybjFweDFmMnk4cmUyb3VxdHFpbDZ5aDhuOXdwNmUwYiZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/101t9QwTM6y5oc/giphy.gif" alt="Cloud Exploit GIF" width="600"/>
</p>

---

## ⚙️ Funcionalidades por Provedor

### 1. AWS (Amazon Web Services)
*   **Detecção de Shadow Admin**: Varredura e análise estática detalhada de políticas de IAM (inline e attached) para encontrar permissões perigosas de escalada de privilégios (ex: `iam:CreateAccessKey`, `iam:PassRole`, `lambda:UpdateFunctionCode`).
*   **Auditoria de IAM Users**: Mapeia todos os usuários da conta, verificando status de MFA, inatividade (data da última senha) e chaves de acesso antigas/não rotacionadas (>90 dias).
*   **Segurança Geral de Conta**: Checagem de MFA e presença de chaves de acesso ativas para o usuário Root da conta, bem como validação da política de senhas (`GetAccountPasswordPolicy`).
*   **Auditoria de Logging (CloudTrail)**: Valida se o CloudTrail está ativo, logging habilitado e cobrindo todas as regiões.
*   **Identity Providers**: Auditoria de provedores SAML e OIDC cadastrados no IAM.

### 2. Azure (Microsoft Azure)
*   **Detecção de Shadow Admin**: Análise de Role Definitions sob a assinatura para mapear privilégios perigosos que permitam escalada de privilégios no ecossistema Azure (ex: `Microsoft.Authorization/roleAssignments/write`, `Microsoft.Compute/virtualMachines/runCommand/action`).
*   **Auditoria de Atribuições (Role Assignments)**: Varredura de todas as atribuições de roles na assinatura, resolvendo os IDs de definições de roles para nomes amigáveis (ex: `Owner`, `Contributor`) e listando os usuários, grupos ou service principals associados.

### 3. GCP (Google Cloud Platform)
*   **Detecção de Shadow Admin**: Varredura de Custom Roles de IAM do projeto para detectar caminhos de impersonificação de contas de serviço e escalação lateral (ex: `iam.serviceAccounts.actAs`, `resourcemanager.projects.setIamPolicy`).
*   **Auditoria de Service Accounts**: Mapeamento de contas de serviço, sinalizando chaves criadas por usuários (`USER_MANAGED`), idade de cada chave (alerta para chaves com mais de 90 dias) e contas desabilitadas com chaves ativas.

---

## 🛠️ Como Compilar e Rodar

### Pré-requisitos
*   Go 1.25+ instalado.
*   Credenciais locais configuradas no ambiente (`aws configure`, `az login` ou `gcloud auth`).

### Compilação
Navegue até o diretório do projeto Go e compile o executável:
```bash
cd squeak-the-cloud
go build -o squeak-audit main.go
```

### Execução
Rode o executável especificando o provedor desejado através da flag `--provider`:

#### AWS
```bash
./squeak-audit --provider aws
```

#### Azure
```bash
export AZURE_SUBSCRIPTION_ID="sua-subscription-id" # Opcional (caso contrário, descobre dinamicamente)
./squeak-audit --provider azure
```

#### GCP
```bash
export GCP_PROJECT_ID="seu-projeto-id" # Opcional (caso contrário, descobre via credenciais JSON)
./squeak-audit --provider gcp
```

---

## 📊 Relatórios e Consolidação

Os resultados individuais de cada recurso auditado são salvos automaticamente em formato JSON dentro do diretório `./results`:
*   `aws_iam_role_*.json`, `aws_iam_user_*.json`, `aws_account_security.json`, `aws_logging_security.json`
*   `azure_role_*.json`, `azure_role_assignment_*.json`
*   `gcp_role_*.json`, `gcp_service_account_*.json`

Ao final de cada execução, o SqueakTheCloud consolida todos os artefatos gerados para aquele provedor no arquivo `./results/consolidated_results.json`.
