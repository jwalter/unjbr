// Full-mesh WebRTC client. Signaling arrives over Server-Sent Events from
// /events and is sent to other peers via POST /signal.

const rtcConfig = { iceServers: [{ urls: "stun:stun.l.google.com:19302" }] };

const $ = (sel) => document.querySelector(sel);
const lobby = $("#lobby");
const room = $("#room");
const grid = $("#grid");
const errorEl = $("#error");
const joinBtn = $("#join");
const micBtn = $("#mic");
const camBtn = $("#cam");

const peers = new Map(); // peer id -> RTCPeerConnection
let myId = null;
let localStream = null;
let events = null;
let sendChain = Promise.resolve();

joinBtn.addEventListener("click", join);
$("#leave").addEventListener("click", leave);
micBtn.addEventListener("click", () => toggle(micBtn, localStream.getAudioTracks(), "Mute", "Unmute"));
camBtn.addEventListener("click", () => toggle(camBtn, localStream.getVideoTracks(), "Stop video", "Start video"));

// If the browser blocked autoplay, any click resumes stalled videos.
document.addEventListener("click", () => {
  for (const video of grid.querySelectorAll("video")) {
    if (video.paused) video.play().catch(() => {});
  }
});

async function join() {
  joinBtn.disabled = true;
  errorEl.hidden = true;
  try {
    localStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
  } catch (err) {
    errorEl.textContent = `Could not access camera or microphone: ${err.message}`;
    errorEl.hidden = false;
    joinBtn.disabled = false;
    return;
  }

  addTile("local", localStream, "You");
  lobby.hidden = true;
  room.hidden = false;

  // Handle events one at a time so each await finishes before the next
  // message touches the same connection.
  let chain = Promise.resolve();
  const handle = (fn) => (e) => {
    const msg = JSON.parse(e.data);
    chain = chain.then(() => fn(msg)).catch((err) => console.error(err));
  };

  events = new EventSource("/events");
  events.addEventListener("welcome", handle(onWelcome));
  events.addEventListener("signal", handle(onSignal));
  events.addEventListener("leave", handle(({ id }) => removePeer(id)));
}

function leave() {
  events?.close();
  events = null;
  for (const id of [...peers.keys()]) removePeer(id);
  localStream?.getTracks().forEach((t) => t.stop());
  localStream = null;
  myId = null;
  grid.replaceChildren();
  resetToggle(micBtn, "Mute");
  resetToggle(camBtn, "Stop video");
  room.hidden = true;
  lobby.hidden = false;
  joinBtn.disabled = false;
}

// The newcomer calls everyone already in the room; existing peers only
// answer. This avoids both sides sending offers at once.
async function onWelcome({ id, peers: others }) {
  // A welcome after a reconnect means we have a new id; start over.
  for (const peerId of [...peers.keys()]) removePeer(peerId);
  myId = id;
  for (const peerId of others) {
    const pc = createPeer(peerId);
    await pc.setLocalDescription();
    send(peerId, "description", pc.localDescription);
  }
}

async function onSignal({ from, type, data }) {
  let pc = peers.get(from);
  if (type === "description") {
    if (!pc) pc = createPeer(from);
    await pc.setRemoteDescription(data);
    if (data.type === "offer") {
      await pc.setLocalDescription();
      send(from, "description", pc.localDescription);
    }
  } else if (type === "candidate" && pc) {
    await pc.addIceCandidate(data);
  }
}

function createPeer(id) {
  const pc = new RTCPeerConnection(rtcConfig);
  for (const track of localStream.getTracks()) pc.addTrack(track, localStream);

  pc.onicecandidate = ({ candidate }) => {
    if (candidate) send(id, "candidate", candidate);
  };
  pc.ontrack = ({ streams: [stream] }) => addTile(id, stream, `Guest ${id.slice(0, 4)}`);
  pc.onconnectionstatechange = () => {
    if (pc.connectionState === "failed") removePeer(id);
  };

  peers.set(id, pc);
  return pc;
}

function removePeer(id) {
  peers.get(id)?.close();
  peers.delete(id);
  document.getElementById(`tile-${id}`)?.remove();
}

// Signaling messages must arrive in order, so POSTs are sent one at a time.
function send(to, type, data) {
  const body = JSON.stringify({ from: myId, to, type, data });
  sendChain = sendChain
    .then(() => fetch("/signal", { method: "POST", headers: { "Content-Type": "application/json" }, body }))
    .catch((err) => console.error("signal failed", err));
}

function addTile(id, stream, label) {
  // ontrack fires once per track; reassigning the same stream would restart
  // loading and abort autoplay.
  const existing = document.getElementById(`tile-${id}`);
  if (existing) {
    const video = existing.querySelector("video");
    if (video.srcObject !== stream) video.srcObject = stream;
    return;
  }

  const tile = document.createElement("figure");
  tile.id = `tile-${id}`;
  tile.className = "tile";

  const video = document.createElement("video");
  video.autoplay = true;
  video.playsInline = true;
  video.srcObject = stream;
  if (id === "local") {
    video.muted = true;
    tile.classList.add("local");
  }

  const caption = document.createElement("figcaption");
  caption.textContent = label;

  tile.append(video, caption);
  grid.append(tile);
}

function toggle(button, tracks, onLabel, offLabel) {
  const off = button.getAttribute("aria-pressed") !== "true";
  tracks.forEach((t) => (t.enabled = !off));
  button.setAttribute("aria-pressed", String(off));
  button.textContent = off ? offLabel : onLabel;
}

function resetToggle(button, label) {
  button.setAttribute("aria-pressed", "false");
  button.textContent = label;
}
