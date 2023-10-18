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

const connect = () => {
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

  pc.addTransceiver('video', { direction: 'recvonly' })
  pc.addTransceiver('audio', { direction: 'recvonly' })
  if (enableTalk.value) {
    pc.addTransceiver('audio', { direction: 'sendonly' })
  }

  pc.addEventListener('icecandidate', (e) => {
    if (e.candidate) {
      ws.value?.send(JSON.stringify({ candidate: e.candidate }))
    }
  })

  pc.addEventListener('connectionstatechange', () => {
    console.log(pc.connectionState)
  })

  pc.addEventListener('track', (e) => {
    console.log('on track', e.track.kind)
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
}

onUnmounted(() => {
  if (ws.value && ws.value.readyState === ws.value.OPEN) {
    console.log('exit due to unmounted')
    ws.value?.close(4500, 'exit')
  }
})
</script>

<template>
  <div>
    <div>
      <input v-model="wsUrl" type="text" placeholder="server, ws://" />
      <input v-model="address" type="text" placeholder="ipc address" />
      <input v-model="username" type="text" placeholder="ipc username" />
      <input v-model="password" type="password" placeholder="ipc password" />
    </div>
    <div>
      <button @click.prevent="connect">connect</button>
    </div>
    <div>
      <video ref="videoEl" muted autoplay></video>
    </div>
  </div>
</template>
