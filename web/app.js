const loginContainer = document.getElementById('login-container');
const viewerContainer = document.getElementById('viewer-container');
const sessionInput = document.getElementById('session-code');
const connectBtn = document.getElementById('connect-btn');
const statusMsg = document.getElementById('status-message');
const canvas = document.getElementById('screen-canvas');
const ctx = canvas.getContext('2d', { alpha: false, desynchronized: true });

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

    dataChannel = pc.createDataChannel('control');
    dataChannel.binaryType = 'blob';

    dataChannel.onopen = () => {
        console.log('Data channel opened');
        dataChannel.send('Hello from Browser!');
    };

    // High-performance hardware-accelerated frame rendering pipeline
    let pendingBlob = null;
    let isRendering = false;

    dataChannel.onmessage = (event) => {
        if (event.data instanceof Blob) {
            pendingBlob = event.data;
            if (!isRendering) {
                isRendering = true;
                requestAnimationFrame(renderFrame);
            }
        }
    };

    async function renderFrame() {
        if (!pendingBlob) {
            isRendering = false;
            return;
        }

        const blob = pendingBlob;
        pendingBlob = null;

        try {
            const bitmap = await createImageBitmap(blob);
            if (canvas.width !== bitmap.width || canvas.height !== bitmap.height) {
                canvas.width = bitmap.width;
                canvas.height = bitmap.height;
            }
            ctx.drawImage(bitmap, 0, 0);
            bitmap.close();
        } catch (err) {
            console.error('Frame decode error:', err);
        }

        if (pendingBlob) {
            requestAnimationFrame(renderFrame);
        } else {
            isRendering = false;
        }
    }

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
    function getNormalizedCoords(e) {
        if (!canvas.width || !canvas.height) return null;
        
        const rect = canvas.getBoundingClientRect();
        const imgRatio = canvas.width / canvas.height;
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

        if (x < 0 || x > renderWidth || y < 0 || y > renderHeight) return null;

        return {
            x: Math.max(0, Math.min(1, x / renderWidth)),
            y: Math.max(0, Math.min(1, y / renderHeight))
        };
    }

    function getButtonName(buttonCode) {
        switch (buttonCode) {
            case 0: return 'left';
            case 1: return 'middle';
            case 2: return 'right';
            default: return 'left';
        }
    }

    function sendEvent(payload) {
        if (dataChannel && dataChannel.readyState === 'open') {
            dataChannel.send(JSON.stringify(payload));
        }
    }

    let lastMove = 0;
    viewerContainer.addEventListener('mousemove', (e) => {
        const coords = getNormalizedCoords(e);
        if (!coords) return;
        const now = Date.now();
        if (now - lastMove > 16) { // ~60fps
            lastMove = now;
            sendEvent({
                type: 'mouse_move',
                x: coords.x,
                y: coords.y
            });
        }
    });

    viewerContainer.addEventListener('mousedown', (e) => {
        const coords = getNormalizedCoords(e);
        if (!coords) return;
        sendEvent({
            type: 'mouse_down',
            button: getButtonName(e.button),
            x: coords.x,
            y: coords.y
        });
    });

    viewerContainer.addEventListener('mouseup', (e) => {
        const coords = getNormalizedCoords(e);
        if (!coords) return;
        sendEvent({
            type: 'mouse_up',
            button: getButtonName(e.button),
            x: coords.x,
            y: coords.y
        });
    });

    viewerContainer.addEventListener('dblclick', (e) => {
        const coords = getNormalizedCoords(e);
        if (!coords) return;
        sendEvent({
            type: 'double_click',
            button: 'left',
            x: coords.x,
            y: coords.y
        });
    });

    viewerContainer.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        const coords = getNormalizedCoords(e);
        if (!coords) return;
        sendEvent({
            type: 'mouse_click',
            button: 'right',
            x: coords.x,
            y: coords.y
        });
    });

    viewerContainer.addEventListener('wheel', (e) => {
        e.preventDefault();
        const coords = getNormalizedCoords(e);
        sendEvent({
            type: 'mouse_scroll',
            delta_x: Math.round(e.deltaX),
            delta_y: Math.round(e.deltaY),
            x: coords ? coords.x : 0,
            y: coords ? coords.y : 0
        });
    }, { passive: false });

    document.addEventListener('keydown', (e) => {
        if (viewerContainer.style.display === 'none') return;
        if (!dataChannel || dataChannel.readyState !== 'open') return;
        
        if (["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Space", "Tab", "Backspace"].includes(e.code)) {
            e.preventDefault();
        }

        sendEvent({
            type: 'key_press',
            key: e.key,
            code: e.code,
            alt_key: e.altKey,
            ctrl_key: e.ctrlKey,
            shift_key: e.shiftKey,
            meta_key: e.metaKey
        });
    });
}
