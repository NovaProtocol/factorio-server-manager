#!/bin/sh
set -eu

# Every credential is required. The manager has to be told who the admin is and
# which keys it runs with, and it has to be told the same thing on every start,
# so an unset variable is a hard failure, never a silent default.
require() {
    if [ -z "$(printenv "$1" 2>/dev/null || true)" ]; then
        echo "error: $1 is required and must be set" >&2
        exit 1
    fi
}

# The cookie key signs sessions: base64 that decodes to 16, 24 or 32 bytes. Check
# it here so a bad key fails with a clear message instead of panicking the
# manager after it has already started.
check_cookie_key() {
    if ! printf '%s' "$COOKIE_ENCRYPTION_KEY" | base64 -d >/dev/null 2>&1; then
        echo "error: COOKIE_ENCRYPTION_KEY must be valid base64" >&2
        exit 1
    fi

    decoded_len=$(printf '%s' "$COOKIE_ENCRYPTION_KEY" | base64 -d 2>/dev/null | wc -c | tr -d '[:space:]')
    case "$decoded_len" in
        16|24|32) ;;
        *)
            echo "error: COOKIE_ENCRYPTION_KEY must decode to 16, 24 or 32 bytes (got $decoded_len)" >&2
            exit 1
            ;;
    esac
}

# conf.json is rewritten from the environment on every start, not only when it is
# missing, so the configured credentials are re-applied after a restart even if
# the file was changed in between.
init_config() {
    require ADMIN_USER
    require ADMIN_PASS
    require RCON_PASS
    require COOKIE_ENCRYPTION_KEY
    check_cookie_key

    # The session cookie is marked Secure by default. A browser discards a Secure
    # cookie that arrives over plain HTTP, so the login appears to succeed and the
    # session is gone on the next request. Set COOKIE_SECURE=false to serve the
    # interface over HTTP; leave it unset behind TLS.
    secure="true"
    if [ "${COOKIE_SECURE:-true}" = "false" ]; then
        secure="false"
    fi

    jq \
      --arg user "$ADMIN_USER" \
      --arg pass "$ADMIN_PASS" \
      --arg rcon "$RCON_PASS" \
      --arg cookie "$COOKIE_ENCRYPTION_KEY" \
      --argjson secure "$secure" \
      '.username = $user
       | .password = $pass
       | .rcon_pass = $rcon
       | .cookie_encryption_key = $cookie
       | .secure = $secure
       | .sq_lite_database_file = "/opt/fsm-data/sqlite.db"
       | .factorio_credentials_file = "/opt/fsm-data/factorio.auth"
       | .log_file = "/opt/fsm-data/factorio-server-manager.log"' \
       /opt/fsm/conf.json >/opt/fsm-data/conf.json

    # conf.json carries the admin password and the RCON password in clear, so it
    # is readable by the manager's account only. The token file and the user
    # database next to it hold secrets of their own and get the same treatment.
    chmod 600 /opt/fsm-data/conf.json
    for secret in factorio.auth sqlite.db; do
        if [ -f "/opt/fsm-data/$secret" ]; then
            chmod 600 "/opt/fsm-data/$secret"
        fi
    done

    echo "Admin user is '$ADMIN_USER'"
    echo "Session cookie secure flag: $secure"
    echo "Rcon password and cookie encryption key are set from the environment"
}

install_game() {
    curl --location "https://www.factorio.com/get-download/${FACTORIO_VERSION}/headless/linux64" \
         --output /tmp/factorio_${FACTORIO_VERSION}.tar.xz
    tar -xf /tmp/factorio_${FACTORIO_VERSION}.tar.xz
    rm /tmp/factorio_${FACTORIO_VERSION}.tar.xz
}

init_config

install_game

cd /opt/fsm && ./factorio-server-manager --conf /opt/fsm-data/conf.json --dir /opt/factorio --port 80
