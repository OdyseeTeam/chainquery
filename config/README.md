#Chainquery Configuration

The Chainquery application comes with a watched configuration that when updated will be updated live in the application. Please see the default[configuration](default/chainqueryconfig.toml)for details. More elaborate documentation is on the way.

The order of precedence for where chainquery looks for a configuration file are `--configpath` flag, `$HOME/`, `.` working directory, and last `./config/default/` for running from src.

Testing