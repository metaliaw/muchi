# Bruno Collections

Open `bruno/Muchi API` in Bruno and select the `Local` Environment.

Run `Create Search` before the Search status, Results, or Cancel Requests.
The Request saves the returned Search ID into the Environment automatically.

The Local Environment expects the API at `http://127.0.0.1:8081` with the
development Bearer Token from `compose.yaml`.

`Production` leaves the Token empty on purpose: it is a Secret Variable, so
Bruno asks for it and never writes it to Disk. The Token lives in Secret
Manager and muchi-api rotates it.

The whole Collection runs from a Terminal:

```bash
npx @usebruno/cli run --env Local -r
```
