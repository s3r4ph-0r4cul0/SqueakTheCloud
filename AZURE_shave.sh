#!/bin/bash

# ================== Azure_Shave ==================

METADATA_URL="http://169.254.169.254/metadata"
API_VERSION="2021-02-01"

BLUE="\e[34m"
GREEN="\e[32m"
RED="\e[31m"
YELLOW="\e[33m"
RESET="\e[0m"

get_metadata() {
    curl -s \
    -H Metadata:true \
    "${METADATA_URL}/instance?api-version=${API_VERSION}"
}

get_access_token() {
    curl -s \
    -H Metadata:true \
    "${METADATA_URL}/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/"
}

export_token() {
    TOKEN_JSON=$1

    export AZURE_ACCESS_TOKEN=$(echo "$TOKEN_JSON" | jq -r .access_token)

    if [[ "$AZURE_ACCESS_TOKEN" == "null" ]]; then
        echo -e "${RED}[X] Falha ao obter token${RESET}"
        exit 1
    fi
}

login_az() {

    cat <<EOF > /tmp/azure_token.json
{
 "accessToken":"$AZURE_ACCESS_TOKEN",
 "subscription":"dummy",
 "tenant":"dummy",
 "tokenType":"Bearer"
}
EOF

}

list_azure_resources() {

    echo -e "${YELLOW}[!] Enumerando recursos Azure${RESET}"

    az vm list -o table

    az vmss list -o table

    az storage account list -o table

    az keyvault list -o table

    az webapp list -o table

    az functionapp list -o table

    az sql server list -o table

    az sql db list -o table

    az aks list -o table

    az container list -o table

    az network vnet list -o table

    az network nsg list -o table

    az network public-ip list -o table

    az network nic list -o table

    az network route-table list -o table

    az network lb list -o table

    az group list -o table

    az resource list -o table

    az role assignment list -o table

    az ad app list

    az ad sp list

    az monitor log-analytics workspace list

    az redis list

}

run_azure_shave() {

    echo -e "${BLUE}[*] Coletando metadados VM${RESET}"

    META=$(get_metadata)

    if [[ -z "$META" ]]; then
        echo -e "${RED}[X] IMDS indisponível${RESET}"
        exit 1
    fi

    echo "$META" | jq

    echo -e "${BLUE}[*] Obtendo token Managed Identity${RESET}"

    TOKEN=$(get_access_token)

    export_token "$TOKEN"

    echo -e "${GREEN}[+] Token obtido${RESET}"

    echo -e "${BLUE}[*] Assumindo identidade Azure${RESET}"

    az login --identity >/dev/null 2>&1

    echo -e "${GREEN}[+] Login efetuado${RESET}"

    SUB=$(az account show \
        --query id \
        -o tsv)

    echo -e "${GREEN}[+] Subscription: ${SUB}${RESET}"

    list_azure_resources

    echo -e "${GREEN}[✓] Enumeração finalizada${RESET}"

}

run_azure_shave
