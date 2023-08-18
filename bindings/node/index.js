// dns.sql - sqlite3 extension that allows querying dns using sql
import {join} from "node:path"
import {fileURLToPath} from "node:url";
import {arch, platform} from "node:process"

const supported = {
  "darwin": {
    "x64": "sqlite-dns-darwin-universal.dylib",
    "arm64": "sqlite-dns-darwin-universal.dylib"
  },
  "linux": {
    "x64": "sqlite-dns-linux-x64.so",
    "arm64": "sqlite-dns-linux-arm64.so"
  }
}

const find = (obj, key) => obj !== null ? obj[key] : null

// Path returns the local path to the correct dynamic library to load using sqlite3_load_extension() function
export function path() {
  let p = find(find(supported, platform), arch)
  if (p === null) {
    throw new Error(`unsupported platform: ${platform}-${arch}`)
  }

  return join(fileURLToPath(new URL(".", import.meta.url)), 'lib', p)
}