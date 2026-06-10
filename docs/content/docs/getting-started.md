---
title: "Instalação e Configuração"
description: "Requisitos de compilação e configuração de credenciais dos provedores de nuvem para o Squeak the Cloud."
weight: 10
---

Este guia descreve como compilar o **Squeak the Cloud** a partir do código-fonte e configurar o seu ambiente de terminal com as credenciais necessárias para auditar a AWS, Azure e GCP.

---

## 🛠️ Requisitos de Compilação

Para compilar e rodar a ferramenta localmente, você precisa ter o **Go 1.25+** instalado em seu sistema operacional. 

Você pode verificar a sua versão instalada do Go com o comando:
```bash
go version
```

---

## 🚀 Compilando o Binário

1. Clone o repositório oficial do projeto:
   ```bash
   git clone https://github.com/s3r4ph-0r4cul0/SqueakTheCloud.git
   cd SqueakTheCloud
   ```

2. Certifique-se de que as dependências do módulo Go estão limpas e prontas:
   ```bash
   go mod tidy
   ```

3. Compile o código-fonte para gerar um binário leve e otimizado:
   ```bash
   go build -o squeak-audit main.go
   ```

Isso gerará o binário executável `squeak-audit` na raiz do projeto.

---

## 🔑 Configuração de Credenciais

O **Squeak the Cloud** interage diretamente com as APIs nativas de cada nuvem usando as sessões e credenciais configuradas localmente no seu terminal. Certifique-se de que a identidade usada possui permissões de leitura (Read-Only/Metadata Reader) para que a enumeração e auditoria ocorram com sucesso.

Configure o acesso no seu ambiente de acordo com o provedor:

### 🟡 AWS (Amazon Web Services)
Você pode configurar as credenciais via CLI padrão da AWS ou exportando variáveis de ambiente:
```bash
# Método 1: Configuração interativa da AWS CLI
aws configure

# Método 2: Definindo variáveis de ambiente diretamente
export AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
export AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
export AWS_DEFAULT_REGION="us-east-1"
```

### 🔵 Microsoft Azure
Certifique-se de ter a ferramenta `az` CLI instalada e autentique-se:
```bash
# Realiza o login interativo no navegador
az login

# (Opcional) Defina a Subscription ativa se possuir mais de uma
export AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
```

### 🟢 GCP (Google Cloud Platform)
Autentique-se utilizando a CLI do gcloud ou aponte para uma chave de Service Account:
```bash
# Método 1: Autenticação padrão de aplicação para APIs locais
gcloud auth application-default login

# Método 2: Utilizando um arquivo JSON de credenciais de Service Account
export GOOGLE_APPLICATION_CREDENTIALS="/caminho/para/sua_chave_gcp.json"

# (Opcional) Defina o ID do projeto GCP ativo
export GCP_PROJECT_ID="target-project-id"
```
