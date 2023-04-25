#! /usr/bin/env bash

set -o errexit   # abort on nonzero exitstatus
set -o nounset   # abort on unbound variable
set -o pipefail  # don't hide errors within pipes
set -x  # debug output feature

###

DIR=$( dirname -- "$( readlink -f -- "$0"; )"; )
PARENT_DIR=$( dirname -- $DIR )
RULES_DIR="$PARENT_DIR/rules"
echo $RULES_DIR

handle_repo() {
    local ARGS=("$@")
    local REPO="${ARGS[0]}"
    local BRANCH_OR_HASH="${ARGS[1]}"
    local RULES="${ARGS[@]:2}"
    
    local REPO_DIR=$(mktemp -d)
    echo $REPO_DIR
    git clone $REPO $REPO_DIR

    pushd "${REPO_DIR}"
    # git pull origin $BRANCH_OR_HASH
    git checkout origin/$BRANCH_OR_HASH
    popd

    for rule in ${RULES[@]}; do
        local TARGET_FILE="$RULES_DIR/$rule"
        local TARGET_FOLDER=$( dirname -- $TARGET_FILE )
        mkdir -p $TARGET_FOLDER
        cp "$REPO_DIR/$rule" $TARGET_FILE
    done
}

rules=(
    "csharp/dotnet/security/use_weak_rng_for_keygeneration.yaml"
    "csharp/dotnet/security/use_ecb_mode.yaml"
    "generic/ci/security/bash-reverse-shell.yaml"
    "go/grpc/security/grpc-client-insecure-connection.yaml"
    "go/jwt-go/security/jwt-none-alg.yaml"
    "go/lang/security/audit/crypto/ssl.yaml"
    "go/lang/security/audit/crypto/tls.yaml"
    "go/lang/security/audit/crypto/use_of_weak_rsa_key.yaml"
    "go/lang/security/audit/net/bind_all.yaml"
    "go/lang/security/injection/tainted-sql-string.yaml"
    "java/java-jwt/security/jwt-hardcode.yaml"
    "java/java-jwt/security/jwt-none-alg.yaml"
    "java/lang/security/audit/blowfish-insufficient-key-size.yaml"
    "java/lang/security/audit/cbc-padding-oracle.yaml"
    "java/lang/security/audit/crypto/des-is-deprecated.yaml"
    "java/lang/security/audit/crypto/desede-is-deprecated.yaml"
    "java/lang/security/audit/crypto/ecb-cipher.yaml"
    "java/lang/security/audit/crypto/gcm-nonce-reuse.yaml"
    "java/lang/security/audit/crypto/no-null-cipher.yaml"
    "java/lang/security/audit/crypto/rsa-no-padding.yaml"
    "java/lang/security/audit/crypto/use-of-md5-digest-utils.yaml"
    "java/lang/security/audit/crypto/use-of-sha1.yaml"
    "java/lang/security/audit/crypto/weak-rsa.yaml"
    "java/lang/security/audit/xxe/documentbuilderfactory-disallow-doctype-decl-false.yaml"
    "java/lang/security/audit/xxe/documentbuilderfactory-external-general-entities-true.yaml"
    "java/lang/security/audit/xxe/documentbuilderfactory-external-parameter-entities-true.yaml"
    "java/spring/security/injection/tainted-file-path.yaml"
    "java/spring/security/injection/tainted-system-command.yaml"
    "javascript/angular/security/detect-angular-sce-disabled.yaml"
    "javascript/express/security/audit/express-libxml-noent.yaml"
    "javascript/express/security/audit/express-open-redirect.yaml"
    "javascript/express/security/audit/express-third-party-object-deserialization.yaml"
    "javascript/express/security/express-jwt-hardcoded-secret.yaml"
    "javascript/jose/security/jwt-hardcode.yaml"
    "javascript/jose/security/jwt-none-alg.yaml"
    "javascript/passport-jwt/security/passport-hardcode.yaml"
    "javascript/sequelize/security/audit/sequelize-injection-express.yaml"
    "kotlin/lang/security/weak-rsa.yaml"
    "php/lang/security/assert-use.yaml"
    "php/lang/security/openssl-cbc-static-iv.yaml"
    "python/django/security/injection/command/subprocess-injection.yaml"
    "python/flask/security/audit/app-run-param-config.yaml"
    "python/flask/security/audit/debug-enabled.yaml"
    "python/flask/security/injection/subprocess-injection.yaml"
    "python/jwt/security/jwt-hardcode.yaml"
    "ruby/lang/security/force-ssl-false.yaml"
    "ruby/lang/security/hardcoded-http-auth-in-controller.yaml"
    "ruby/lang/security/hardcoded-secret-rsa-passphrase.yaml"
    "ruby/lang/security/insufficient-rsa-key-size.yaml"
    "scala/jwt-scala/security/jwt-scala-hardcode.yaml"
    "scala/lang/security/audit/documentbuilder-dtd-enabled.yaml"
    "scala/lang/security/audit/rsa-padding-set.yaml"
    "scala/lang/security/audit/sax-dtd-enabled.yaml"
    "scala/lang/security/audit/xmlinputfactory-dtd-enabled.yaml"
    "scala/play/security/tainted-slick-sqli.yaml"
    "scala/play/security/tainted-sql-from-http-request.yaml"
    "scala/scala-jwt/security/jwt-hardcode.yaml"
    "terraform/aws/security/aws-elasticsearch-insecure-tls-version.yaml"
    "terraform/azure/security/appservice/appservice-use-secure-tls-policy.yaml"
    "yaml/github-actions/security/github-script-injection.yaml"
    "terraform/aws/security/aws-config-aggregator-not-all-regions.yaml"
    "java/lang/security/audit/crypto/use-of-aes-ecb.yaml"
    "java/lang/security/audit/crypto/use-of-blowfish.yaml"
    "java/lang/security/audit/crypto/use-of-default-aes.yaml"
    "java/lang/security/audit/crypto/use-of-rc2.yaml"
    "java/lang/security/audit/crypto/use-of-rc4.yaml"
)

handle_repo "https://github.com/returntocorp/semgrep-rules.git" "release" ${rules[@]}
