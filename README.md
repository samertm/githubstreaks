# GitHubStreaks [![Build Status](https://travis-ci.org/samertm/githubstreaks.svg)](https://travis-ci.org/samertm/githubstreaks)

# Setup

```bash
$ pacman -S postgres
$ sudo -i -u postgres initdb --locale en_US.UTF-8 -E UTF8 -D '/var/lib/postgres/data'
$ sudo systemctl start postgresql
$ sudo systemctl enable postgresql
$ sudo -i -u postgres createuser --interactive <your-system-username>
$ createdb githubstreaks
```

# Idea

Can I create a dockerfile for a dev postgres thingy to run?

# TODO
 - Make Setup accurate.
 - Get linter/test thingy for django templates.
