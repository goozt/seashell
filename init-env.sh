download_go() {
    SOURCE_FILE="$(pwd)/tmp/go.tar.gz"
    test -f "$SOURCE_FILE" && return 0
    GO_URL="https://go.dev/dl/go1.18.10.linux-amd64.tar.gz"
    wget -q $GO_URL -O "$SOURCE_FILE" && return 0
    return 1
}

install_go() {
    echo "Installing Go in current folder"
    mkdir -p tmp bin && \
    download_go || return 1
    tar -C tmp -xzf tmp/go.tar.gz && \
    install "$(pwd)/tmp/go/bin/go" bin/go && \
    echo -e "\033[1A\033[KGo installation complete" && return 0
    echo "Go installation failed" && return 1
}

clean() {
    rm -rf tmp bin && echo "cleaned go binary and temp directories"
}

is_go_installed() {
    test -d "$(pwd)/tmp/go" || return 1
    ("$(pwd)/bin/go" version 2>/dev/null | grep -q "go1.18.10") || return 1
    return 0
}

is_go_compatible() {
    (go version 2>/dev/null | grep -q "go1.18") || return 1
    return 0
}

main() {
    if !(is_go_installed); then
        is_go_compatible && return 0 || \
        echo "Go version 1.18 not found in system"

        install_go && return 0
        clean && return 1
    fi
}

main $@
