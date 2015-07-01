# Setup

$ pacman -S postgres
$ sudo -i -u postgres initdb --locale en_US.UTF-8 -E UTF8 -D '/var/lib/postgres/data'
$ sudo systemctl start postgresql
$ sudo systemctl enable postgresql
$ sudo -i -u postgres createuser --interactive <your-system-username>
$ createdb githubstreaks
