# Exposure sites notifier

Checks Victorian exposure site data, filters for a suburb of interest, notifies if new exposure sites detected

```
docker run -e TOKEN=__meeiot_token__ ghcr.io/codingric/exposure_sites
```

## Configuration/Requirements

### Twilio

- Create a twilio account
- Generate API token

### State Key Store

- Generate a new token here: https://www.meeiot.org/?p=start.php
- Export to environment variable
```
export TOKEN={token}
```
- Seed your configuration
```
export STATEDATA=$(echo -n '{"hash":null,"mobile":"+614xxxxxxxx","sid":"__twilio_sid__","token":"__twilio_token__","suburb":"Melbourne"}' | base64)
curl https://www.meeiot.org/put/$TOKEN/state=$STATEDATA
```

