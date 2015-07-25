var debug = true;

export default function debuglog() {
  if (debug) {
    console.log.apply(console, arguments);
  }
}
