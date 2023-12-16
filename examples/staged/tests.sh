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
    terraform init -upgrade # in case version was updated
    terraform apply -state=../terraform.tfstate -auto-approve # using state from parent dir
    cd ..
done
set +e