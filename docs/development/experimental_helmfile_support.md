# Experimental Helmfile Support

As a proof of concept, the dogu operator should be able to install Helmfiles from V2 Dogus that have the `"kind": "Helmfile"` property in their `Dogu.json`.

For now, only the openDesk Helmfile will be supported.

## Dogu.json

See [Dogu.json](openDesk_dogu.json) for more information.

Pushing the dogu.json:
```shell
curl --fail-with-body --basic --request PUT \
    --user "$DOGU_REGISTRY_USER" \
    --header "Content-Type: application/json" \
    --data-binary @openDesk_dogu.json \
    "https://dogu.cloudogu.com/api/v2/dogus/internal/opendesk"
```

Deleting the dogu.json:
```shell
curl --fail-with-body --basic --request DELETE \
    --user "$DOGU_REGISTRY_USER" \
    "https://dogu.cloudogu.com/api/v2/dogus/internal/opendesk/1.18.2-1"
```