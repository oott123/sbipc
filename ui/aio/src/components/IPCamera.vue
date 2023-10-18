<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import { useRememberRef } from '../setups/useRememberRef'

const ws = ref<WebSocket>()
const wsConnected = ref(false)
const peerConnection = ref<RTCPeerConnection>()

const wsUrl = useRememberRef('sbipcWsUrl', '')
const enableTalk = useRememberRef('sbipcEnableTalk', false)
const address = useRememberRef('sbipcAddress', '')
const username = useRememberRef('sbipcUsername', '')
const password = useRememberRef('sbipcPassword', '')

const videoEl = ref<HTMLVideoElement>()
const videoStream = ref<MediaStream>()
const audioStream = ref<MediaStream>()
const talkChannel = ref<RTCDataChannel>()

const connect = async () => {
  console.log('start connect')
  videoStream.value = undefined
  audioStream.value = undefined

  ws.value = new WebSocket(wsUrl.value)
  ws.value.addEventListener('open', () => {
    wsConnected.value = true
    ws.value!.send(
      JSON.stringify({
        open: {
          address: address.value,
          username: username.value,
          password: password.value,
          enableTalk: enableTalk.value,
        },
      }),
    )
  })
  ws.value.addEventListener('close', (e) => {
    console.log('close', e.code, e.reason)
    wsConnected.value = false
  })
  ws.value.addEventListener('message', (e) => {
    const data = JSON.parse(e.data)
    if (data.sessionDescription) {
      if (peerConnection.value) {
        peerConnection.value.setRemoteDescription(data.sessionDescription)
        peerConnection.value.setLocalDescription().then(() => {
          ws.value!.send(JSON.stringify({ sessionDescription: peerConnection.value!.localDescription }))
        })
      }
    } else if (data.candidate) {
      peerConnection.value!.addIceCandidate(data.candidate)
    } else if (data.error) {
      console.error(data.error.message)
    }
  })

  const pc = new RTCPeerConnection()
  peerConnection.value = pc

  pc.addEventListener('icecandidate', (e) => {
    if (e.candidate) {
      ws.value?.send(JSON.stringify({ candidate: e.candidate }))
    }
  })

  pc.addEventListener('connectionstatechange', () => {
    console.log(pc.connectionState)
  })

  pc.addEventListener('track', (e) => {
    if (e.track.kind === 'video') {
      videoStream.value = e.streams[0]
    } else if (e.track.kind === 'audio') {
      audioStream.value = e.streams[0]
    }

    if (videoStream.value) {
      if (audioStream.value) {
        for (const track of audioStream.value.getAudioTracks()) {
          videoStream.value.addTrack(track)
        }
      }
      videoEl.value!.srcObject = videoStream.value
    }
  })

  pc.addEventListener('datachannel', (e) => {
    console.log('on channel', e.channel.label)
    if (e.channel.label === 'talk') {
      talkChannel.value = e.channel
    }
  })
}

const record = async () => {
  navigator.mediaDevices
    .getUserMedia({ audio: { sampleRate: 8000, sampleSize: 16 } })
    .then((stream) => {
      const ac = new AudioContext({
        sampleRate: 8000,
        latencyHint: 'interactive',
      })
      const source = ac.createMediaStreamSource(stream)
      const dest = ac.createMediaStreamDestination()

      const scriptProcessor = ac.createScriptProcessor(256, 1, 1)
      scriptProcessor.onaudioprocess = (e) => {
        const arr = e.inputBuffer.getChannelData(0)
        const alaw = new Uint8Array([...arr].map((x) => encodeSample(Math.round(x * 32767))))
        talkChannel.value?.send(alaw)
      }

      source.connect(scriptProcessor).connect(dest)
    })
    .catch(console.error)
}

onUnmounted(() => {
  if (ws.value && ws.value.readyState === ws.value.OPEN) {
    console.log('exit due to unmounted')
    ws.value?.close(4500, 'exit')
  }
})

/** @type {!Array<number>} */
const LOG_TABLE = [
  1, 1, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 6, 6, 6, 6, 6, 6, 6,
  6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
  7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
  7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
]

/**
 * Encode a 16-bit linear PCM sample as 8-bit A-Law.
 * @param {number} sample A 16-bit PCM sample
 * @return {number}
 */
function encodeSample(sample: any) {
  /** @type {number} */
  let compandedValue
  sample = sample == -32768 ? -32767 : sample
  /** @type {number} */
  let sign = (~sample >> 8) & 0x80
  if (!sign) {
    sample = sample * -1
  }
  if (sample > 32635) {
    sample = 32635
  }
  if (sample >= 256) {
    /** @type {number} */
    let exponent = LOG_TABLE[(sample >> 8) & 0x7f]
    /** @type {number} */
    let mantissa = (sample >> (exponent + 3)) & 0x0f
    compandedValue = (exponent << 4) | mantissa
  } else {
    compandedValue = sample >> 4
  }
  return compandedValue ^ (sign ^ 0x55)
}
</script>

<template>
  <div>
    <div>
      <input v-model="wsUrl" type="text" placeholder="server, ws://" />
      <input v-model="address" type="text" placeholder="ipc address" />
      <input v-model="username" type="text" placeholder="ipc username" />
      <input v-model="password" type="password" placeholder="ipc password" />
      <label><input v-model="enableTalk" type="checkbox" /> enable talk</label>
    </div>
    <div>
      <button @click.prevent="connect">connect</button>
      <button @click.prevent="record">record</button>
    </div>
    <div>
      <video ref="videoEl" muted autoplay width="640" height="360"></video>
    </div>
  </div>
</template>
