const loginContainer = document.getElementById('login-container');
const viewerContainer = document.getElementById('viewer-container');
const sessionInput = document.getElementById('session-code');
const connectBtn = document.getElementById('connect-btn');
const statusMsg = document.getElementById('status-message');
const screenImg = document.getElementById('screen');

let ws;
let pc;
let dataChannel;
let sessionCode = '';

connectBtn.addEventListener('click', () => {
    sessionCode = sessionInput.value.trim();
    if (!sessionCode) {
        statusMsg.textContent = 'Please enter a session code.';
        return;
    }
    connect();
});

function setStatus(msg, isError = false) {
    statusMsg.textContent = msg;
    statusMsg.style.color = isError ? 'var(--error)' : '#94a3b8';
}

function connect() {
    setStatus('Connecting to signaling server...');
    connectBtn.disabled = true;

    // Use current host for websocket
    const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProto}//${window.location.host}/ws`;
    
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        setStatus('Requesting connection to host...');
        ws.send(JSON.stringify({
            type: 'connection_request',
            session: sessionCode
        }));
    };

    ws.onmessage = async (event) => {
        const msg = JSON.parse(event.data);

        switch (msg.type) {
            case 'error':
                setStatus(msg.payload, true);
                connectBtn.disabled = false;
                ws.close();
                break;
            
            case 'connection_response':
                if (msg.payload === 'accepted') {
                    setStatus('Connection accepted! Setting up WebRTC...');
                    setupWebRTC();
                } else {
                    setStatus('Connection rejected by host.', true);
                    connectBtn.disabled = false;
                    ws.close();
                }
                break;
            
            case 'signal':
                handleSignal(msg.signal);
                break;
        }
    };

    ws.onerror = () => {
        setStatus('Failed to connect to signaling server.', true);
        connectBtn.disabled = false;
    };
}

async function setupWebRTC() {
    const configuration = {
        iceServers: [
            { urls: 'stun:stun.l.google.com:19302' },
            { urls: 'stun:stun1.l.google.com:19302' },
            { urls: 'stun:stun2.l.google.com:19302' },
            { urls: 'stun:global.stun.twilio.com:3478' }
        ]
    };

    pc = new RTCPeerConnection(configuration);

    pc.onicecandidate = (event) => {
        if (event.candidate) {
            ws.send(JSON.stringify({
                type: 'signal',
                session: sessionCode,
                signal: JSON.stringify(event.candidate)
            }));
        }
    };

    pc.onconnectionstatechange = () => {
        if (pc.connectionState === 'connected') {
            loginContainer.style.display = 'none';
            viewerContainer.style.display = 'flex';
            setupInputHandlers();
        } else if (pc.connectionState === 'disconnected' || pc.connectionState === 'failed') {
            alert('Disconnected from host.');
            window.location.reload();
        }
    };

    // We are the viewer (remote), so we create the data channel
    dataChannel = pc.createDataChannel('control');
    dataChannel.binaryType = 'blob';

    dataChannel.onopen = () => {
        console.log('Data channel opened');
        dataChannel.send('Hello from Browser!');
    };

    let activeUrl = null;
    dataChannel.onmessage = (event) => {
        if (event.data instanceof Blob) {
            if (activeUrl) {
                URL.revokeObjectURL(activeUrl);
            }
            activeUrl = URL.createObjectURL(event.data);
            screenImg.src = activeUrl;
        }
    };

    try {
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        ws.send(JSON.stringify({
            type: 'signal',
            session: sessionCode,
            signal: JSON.stringify(pc.localDescription)
        }));
    } catch (err) {
        console.error('Error creating offer:', err);
        setStatus('Failed to create WebRTC offer.', true);
    }
}

async function handleSignal(signalStr) {
    try {
        const signal = JSON.parse(signalStr);
        if (signal.type === 'answer') {
            await pc.setRemoteDescription(new RTCSessionDescription(signal));
        } else if (signal.candidate) {
            await pc.addIceCandidate(new RTCIceCandidate(signal));
        }
    } catch (err) {
        console.error('Error handling signal:', err);
    }
}

function setupInputHandlers() {
    function sendMouse(type, e) {
        if (dataChannel.readyState !== 'open') return;
        if (!screenImg.naturalWidth) return;
        
        const rect = screenImg.getBoundingClientRect();
        const imgRatio = screenImg.naturalWidth / screenImg.naturalHeight;
        const rectRatio = rect.width / rect.height;
        
        let renderWidth = rect.width;
        let renderHeight = rect.height;
        let offsetX = 0;
        let offsetY = 0;

        if (rectRatio > imgRatio) {
            renderWidth = rect.height * imgRatio;
            offsetX = (rect.width - renderWidth) / 2;
        } else {
            renderHeight = rect.width / imgRatio;
            offsetY = (rect.height - renderHeight) / 2;
        }

        let x = e.clientX - rect.left - offsetX;
        let y = e.clientY - rect.top - offsetY;

        if (x < 0 || x > renderWidth || y < 0 || y > renderHeight) return;

        let normX = x / renderWidth;
        let normY = y / renderHeight;

        dataChannel.send(JSON.stringify({ type: type, x: normX, y: normY }));
    }

    let lastMove = 0;
    document.addEventListener('mousemove', (e) => {
        if (viewerContainer.style.display === 'none') return;
        const now = Date.now();
        if (now - lastMove > 16) {
            lastMove = now;
            sendMouse('mouse_move', e);
        }
    });

    document.addEventListener('mousedown', (e) => {
        if (viewerContainer.style.display === 'none') return;
        sendMouse('mouse_click', e);
    });

    document.addEventListener('keydown', (e) => {
        if (viewerContainer.style.display === 'none') return;
        if (dataChannel.readyState !== 'open') return;
        
        // Prevent default browser actions for common keys to avoid scrolling/refreshing
        if (["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Space", "Enter"].includes(e.code)) {
            e.preventDefault();
        }

        dataChannel.send(JSON.stringify({ type: 'key_press', key: e.key }));
    });
}
