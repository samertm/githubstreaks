// Routes

export function getGID() {
  var u = new URL(document.URL);
  var re = new RegExp("/group/(\\d+)");
  var match = re.exec(u.pathname);
  return Number(match[1]);
}

export function groupRefresh(gid) {
  return "/group/" + gid + "/refresh";
}
