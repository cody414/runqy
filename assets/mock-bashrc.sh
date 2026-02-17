#!/bin/bash
runqy() {
    case "$1 $2" in
        "serve ")
            /tmp/demo-v8/mock-server.sh
            ;;
        "config create")
            /tmp/demo-v8/mock-config-create.sh
            ;;
        "queue list")
            /tmp/demo-v8/mock-queue-list.sh
            ;;
        "worker list")
            /tmp/demo-v8/mock-worker-list.sh
            ;;
        "task enqueue")
            /tmp/demo-v8/mock-enqueue-cli.sh
            ;;
        "task get")
            /tmp/demo-v8/mock-task-get.sh
            ;;
        *)
            echo "runqy $@"
            ;;
    esac
}

runqy-worker() {
    /tmp/demo-v8/mock-worker.sh
}

curl() {
    /tmp/demo-v8/mock-enqueue-rest.sh
}

python3() {
    /tmp/demo-v8/mock-enqueue-python.sh
}

closing() {
    /tmp/demo-v8/mock-closing.sh
}

export -f runqy runqy-worker curl python3 closing
export PS1="\[\033[1;32m\]$\[\033[0m\] "
