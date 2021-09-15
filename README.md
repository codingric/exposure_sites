# Exposure sites notifier

Checks Victorian exposure site data, filters for a suburb of interest, notifies if new exposure sites detected

## Requirements

### Twilio

- Create a twilio account
- Generate API token

### State Key Store

- Generate a new token here: https://www.meeiot.org/?p=start.php
- Export to environment variable
```
export TOKEN=__meeiot_token__
```
- Seed your configuration
```
docker run -it ghcr.io/codingric/exposure_sites config
```

### Running a check

```
docker run -e TOKEN ghcr.io/codingric/exposure_sites
```

