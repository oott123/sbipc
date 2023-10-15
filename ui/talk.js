const fields = ['address', 'username', 'password']
for (const field of fields) {
  const el = document.querySelector(`#talkform [name="${field}"]`)
  const key = `sbipc_${field}`
  if (localStorage[key]) {
    el.value = localStorage[key]
  }

  el.addEventListener('change', (e) => {
    localStorage[key] = e.target.value
  })
}

document.getElementById('talkform').addEventListener('submit', (e) => {
  e.preventDefault()
  const data = new FormData(e.target)
  const search = new URLSearchParams()
  search.append('address', data.get('address'))
  search.append('username', data.get('username'))
  search.append('password', data.get('password'))
  const urlBuilder = new URL('/talk', location.href)
  urlBuilder.search = search.toString()
  const wsUrl = urlBuilder.toString().replace(/^http/, 'ws')
  addLog(`Connecting`)
  const ws = new WebSocket(wsUrl)
  ws.onopen = () => {
    addLog('Connected')
  }
  ws.onclose = () => {
    addLog('Disconnected')
  }

  navigator.mediaDevices.getUserMedia({ audio: { sampleRate: 8000, sampleSize: 16 } })
  .then(stream => {
    const ac = new AudioContext({
      sampleRate: 8000,
      latencyHint: "interactive",
    })
    const source = ac.createMediaStreamSource(stream)
    const dest = ac.createMediaStreamDestination()

    const scriptProcessor = ac.createScriptProcessor(256, 1, 1)
    scriptProcessor.onaudioprocess = (e) => {
      const arr = e.inputBuffer.getChannelData(0)
      const alaw = new Uint8Array([...arr].map(x => encodeSample(Math.round(x * 32767))))
      if (ws.readyState === ws.OPEN) {
        ws.send(alaw)
      }
    }

    source.connect(scriptProcessor).connect(dest)

    document.getElementById('stop-btn').onclick = () => {
      stream.getTracks().forEach((track) => {
        track.stop()
      })
      ws.close()
    }
  })
  .catch(console.error)
})

/** @type {!Array<number>} */
const LOG_TABLE = [
  1,1,2,2,3,3,3,3,4,4,4,4,4,4,4,4,5,5,5,5,5,5,5,5,5,5,5,5,5,5,5,5, 
  6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6,6, 
  7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7, 
  7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7,7 
];

/**
 * Encode a 16-bit linear PCM sample as 8-bit A-Law.
 * @param {number} sample A 16-bit PCM sample
 * @return {number}
 */
function encodeSample(sample) {
  /** @type {number} */
  let compandedValue; 
  sample = (sample == -32768) ? -32767 : sample;
  /** @type {number} */
  let sign = ((~sample) >> 8) & 0x80; 
  if (!sign) {
    sample = sample * -1; 
  }
  if (sample > 32635) {
    sample = 32635; 
  }
  if (sample >= 256)  {
    /** @type {number} */
    let exponent = LOG_TABLE[(sample >> 8) & 0x7F];
    /** @type {number} */
    let mantissa = (sample >> (exponent + 3) ) & 0x0F; 
    compandedValue = ((exponent << 4) | mantissa); 
  } else {
    compandedValue = sample >> 4; 
  } 
  return compandedValue ^ (sign ^ 0x55);
}

function addLog(text) {
  document.getElementById('log').value += '\n' + text
  document.getElementById('log').scrollTop = 9999999999
}
