# Oink!
## A lightweight DDNS client for [porkbun](https://porkbun.com)

> NOTE: **Oink!** is in BETA. If you encounter any bugs please report them at https://github.com/RLado/Oink!

**Oink!** is an unofficial DDNS client for porkbun.com built in Go. **Oink!** only depends on Go's standard library.

### How to install
You can install **Oink!** using the official snap package by ... [To Do]

### How to setup
The setup process is simple:

- If installed correctly you should find **Oink!**'s configuration file in */etc/oink_ddns/config.json*. Open the file with your text editor of choice.
- In the configuration file you should find the following contents that must be filled in:
> ⚠️ In case you do not already have an API key, you will need to request one at: https://porkbun.com/account/api
```json
{
    "secretapikey": "<your secret api key here>",
    "apikey": "<your api key here>",
    "domain": "<your domain here>",
    "subdomain": "<your subdomain here>",
    "ttl": 600,
    "priority": 0,
    "interval": 300
}
```
- Enable and start the service using `systemd`
> ⚠️ Make sure to **enable** API ACCESS in your porkbun domain's control panel
```bash
systemctl enable oink_ddns
systemctl start oink_ddns
```
- You are done! Your domain DNS record should update automatically

