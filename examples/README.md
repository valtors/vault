# vault examples

## basic usage

```bash
# run a command in a sandbox
vault run -- echo "hello from sandbox"

# with timeout
vault run -timeout 60 -- python script.py

# allow specific directory access
vault run -allow /home/user/project -- npm test

# allow network to specific hosts
vault run -net-allow github.com -net-allow npmjs.org -- npm install
```

## running an agent

```bash
# run claude code in a sandbox
vault run -allow /home/user/myproject -- claude-code

# the agent:
# - can read files in /home/user/myproject
# - cannot read ~/.ssh, ~/.aws, ~/.env
# - cannot see real env vars (all secrets stripped)
# - cannot call arbitrary hosts (network policy)
# - all actions logged to sqlite audit log
```

## api server

```bash
# start the api
vault serve -port 9090

# create a sandbox
curl -X POST localhost:9090/sandboxes \
  -d '{"command":"python","args":["train.py"],"timeout":300}'

# list sandboxes
curl localhost:9090/sandboxes

# get audit logs
curl localhost:9090/sandboxes/1/logs

# kill a sandbox
curl -X POST localhost:9090/sandboxes/1/kill
```

## network policy

```bash
# deny all by default, allow specific hosts
vault run -net-default deny -net-allow github.com -net-allow pypi.org -- pip install

# allow all but deny specific hosts
vault run -net-default allow -net-deny internal.company.com -- npm test
```

## what the agent sees

the agent runs in a sandbox where:
- `~/.ssh/` does not exist
- `~/.aws/` does not exist
- `~/.env` does not exist
- env vars like `GITHUB_TOKEN`, `OPENAI_API_KEY`, `DATABASE_URL` are empty
- filesystem writes go to a temp overlay, not your real disk
- network connections to non-allowlisted hosts are refused
- every action is logged to `~/.vault/audit.db`
