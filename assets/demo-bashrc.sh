#!/bin/bash
# Mock commands for demo recording
runqy() {
    case "$1 $2" in
        "serve ")
            /tmp/demo-server.sh
            ;;
        "task enqueue")
            /tmp/demo-cli.sh
            ;;
        "queue list")
            /tmp/demo-qlist.sh
            ;;
        *)
            echo "runqy $@"
            ;;
    esac
}

runqy-worker() {
    /tmp/demo-worker.sh
}

curl() {
    /tmp/demo-rest.sh
}

python3() {
    /tmp/demo-python.sh
}

export -f runqy runqy-worker curl python3
export PS1="\[\033[1;32m\]$\[\033[0m\] "
