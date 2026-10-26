// Runs after `next build` (output: "standalone"). The server in .next-build/standalone serves public/ and the static
// assets only once they are copied next to it; then it is moved to a new build/server-<time>, and build/server.txt names
// it for the Go backend. A running server keeps its directory locked on Windows, so every build gets a new one and the
// backend removes the old ones once it has stopped their server. Last it writes build/index.html, which CasWAF looks
// for to tell whether the site has been built; without it CasWAF would build again on every round.
import fs from "node:fs";
import path from "node:path";

const webDir = path.resolve(import.meta.dirname, "..");
const distDir = path.join(webDir, ".next-build");
const standaloneDir = path.join(distDir, "standalone");
const buildDir = path.join(webDir, "build");

if (!fs.existsSync(path.join(standaloneDir, "server.js"))) {
  throw new Error(`${standaloneDir}/server.js does not exist`);
}

fs.cpSync(path.join(webDir, "public"), path.join(standaloneDir, "public"), { recursive: true });
fs.cpSync(path.join(distDir, "static"), path.join(standaloneDir, ".next-build", "static"), { recursive: true });

const serverName = `server-${Date.now()}`;
fs.mkdirSync(buildDir, { recursive: true });
fs.renameSync(standaloneDir, path.join(buildDir, serverName));
fs.writeFileSync(path.join(buildDir, "server.txt"), serverName);

fs.writeFileSync(path.join(buildDir, "index.html"), "<!-- built; the pages are served by the server named in build/server.txt -->\n");
console.log(`prepared build/${serverName}`);
