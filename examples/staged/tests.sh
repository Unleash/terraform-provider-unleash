#!/usr/bin/env bash
set -e
# STAGES can be defined via env variable, e.g.: `make STAGES=stage_3` will just run stage 3
if [ -z "$STAGES" ]; then
    STAGES=$(ls | grep stage | sort)
fi
for STAGE in ${STAGES}; do
    echo "================================================================="
    echo "======================= Applying ${STAGE} ========================"
    echo "================================================================="
    cd ${STAGE}
    if [ ! -f ../terraform.tfstate ]; then
        terraform init -upgrade
    else
        terraform init
    fi
    terraform apply -state=../terraform.tfstate -auto-approve # using state from parent dir
    cd ..
done
set +e