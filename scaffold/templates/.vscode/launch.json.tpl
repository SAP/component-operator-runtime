{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Operator",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}",
            "args": [
                "--kubeconfig=${workspaceFolder}/tmp/kubeconfig",
                {{- if or .validatingWebhookEnabled .mutatingWebhookEnabled }}
                "--webhook-tls-directory=${workspaceFolder}/tmp/ssl",
                {{- end }}
                "--zap-log-level=debug"
            ]
        }
    ]
}