(function () {
    // Create and inject Styles
    const style = document.createElement('style');
    style.innerHTML = `
        #ai-chat-widget { position: fixed; bottom: 20px; right: 20px; z-index: 9999; font-family: 'Inter', sans-serif; }
        #ai-chat-toggle { width: 60px; height: 60px; border-radius: 50%; background: #8b5cf6; color: white; border: none; box-shadow: 0 4px 15px rgba(0,0,0,0.3); cursor: pointer; display: flex; align-items: center; justify-content: center; transition: transform 0.3s; }
        #ai-chat-window { display: none; position: absolute; bottom: 80px; right: 0; width: 350px; height: 500px; background: #1a1d24; border: 1px solid #2a2d35; border-radius: 12px; box-shadow: 0 10px 40px rgba(0,0,0,0.4); flex-direction: column; overflow: hidden; backdrop-filter: blur(10px); color: white; }
        .ai-header { padding: 15px; background: #8b5cf6; color: white; font-weight: 700; display: flex; justify-content: space-between; align-items: center; }
        .ai-messages { flex: 1; overflow-y: auto; padding: 15px; display: flex; flex-direction: column; gap: 10px; }
        .ai-input-area { padding: 15px; border-top: 1px solid #2a2d35; background: rgba(0,0,0,0.2); display: flex; gap: 10px; }
        .ai-input { flex: 1; background: #0f1115; border: 1px solid #2a2d35; border-radius: 6px; padding: 8px 12px; color: white; font-size: 0.85rem; outline: none; }
        .ai-send { background: #8b5cf6; color: white; border: none; border-radius: 6px; padding: 0 15px; cursor: pointer; }
        .msg-user { align-self: flex-end; background: #8b5cf6; color: white; padding: 10px; border-radius: 12px 12px 0 12px; font-size: 0.85rem; max-width: 80%; }
        .msg-ai { align-self: flex-start; background: rgba(255,255,255,0.05); color: #f8fafc; padding: 10px; border-radius: 12px 12px 12px 0; font-size: 0.85rem; max-width: 80%; }
        .typing { color: #94a3b8; font-size: 0.8rem; margin-top: 5px; }
    `;
    document.head.appendChild(style);

    // Create Widget
    const widget = document.createElement('div');
    widget.id = 'ai-chat-widget';
    widget.innerHTML = `
        <button id="ai-chat-toggle"><i class="fa-solid fa-robot" style="font-size: 1.5rem;"></i></button>
        <div id="ai-chat-window">
            <div class="ai-header">
                <span><i class="fa-solid fa-microchip"></i> Observability AI</span>
                <i class="fa-solid fa-xmark" id="ai-chat-close" style="cursor: pointer;"></i>
            </div>
            <div id="ai-chat-messages" class="ai-messages">
                <div class="msg-ai">Hi! I'm your Observability Assistant. Ask me anything about system health, metrics, or alerts. You can also type an Incident ID (e.g., inc-a1b2c3d) to have me analyze it!</div>
            </div>
            <div class="ai-input-area">
                <input type="text" id="ai-chat-input" class="ai-input" placeholder="Ask a question...">
                <button id="ai-chat-send" class="ai-send"><i class="fa-solid fa-paper-plane"></i></button>
            </div>
        </div>
    `;
    document.body.appendChild(widget);

    const toggle = document.getElementById('ai-chat-toggle');
    const window = document.getElementById('ai-chat-window');
    const close = document.getElementById('ai-chat-close');
    const input = document.getElementById('ai-chat-input');
    const send = document.getElementById('ai-chat-send');
    const messages = document.getElementById('ai-chat-messages');

    toggle.onclick = () => {
        window.style.display = window.style.display === 'none' ? 'flex' : 'none';
        toggle.style.transform = window.style.display === 'flex' ? 'scale(0.9) rotate(90deg)' : 'scale(1)';
    };

    close.onclick = () => {
        window.style.display = 'none';
        toggle.style.transform = 'scale(1)';
    };

    const addMessage = (text, isUser = false) => {
        const msg = document.createElement('div');
        msg.className = isUser ? 'msg-user' : 'msg-ai';
        msg.innerText = text;
        messages.appendChild(msg);
        messages.scrollTop = messages.scrollHeight;
    };

    const handleSend = async () => {
        const text = input.value.trim();
        if (!text) return;

        addMessage(text, true);
        input.value = '';

        const typing = document.createElement('div');
        typing.className = 'typing';
        typing.innerText = "Analyzing system data...";
        messages.appendChild(typing);
        messages.scrollTop = messages.scrollHeight;

        // Context Enrichment: Find any UUIDs in the message and attach relevant data
        let context = "";
        // Match inc-xxxxx or just xxxxx if preceded by "incident"
        const incidentMatch = text.match(/inc-[a-z0-9]{4,12}/i) || text.match(/(?:incident\s+)([a-z0-9]{7,10})/i);
        const targetMatch = text.match(/trg-[a-z0-9]{4,12}/i) || text.match(/(?:target\s+)([a-z0-9]{7,10})/i);

        if (incidentMatch) {
            let uuid = incidentMatch[0].toLowerCase();
            if (incidentMatch[1]) uuid = 'inc-' + incidentMatch[1].toLowerCase(); // Handle missing prefix

            const uuidMap = JSON.parse(localStorage.getItem('hawkeye_uuid_map') || '{}');
            const inc = uuidMap[uuid] || Object.values(uuidMap).find(i => i.uuid.includes(uuid.replace('inc-', '')));

            if (inc) {
                context += `\n[CRITICAL INCIDENT DATA FOUND for ${uuid}: ${JSON.stringify(inc)}]`;
                console.log("AI Context Found for Incident:", uuid);
            }
        }

        if (targetMatch) {
            let uuid = targetMatch[0].toLowerCase();
            if (targetMatch[1]) uuid = 'trg-' + targetMatch[1].toLowerCase();

            const targetUuidMap = JSON.parse(localStorage.getItem('hawkeye_target_uuid_map') || '{}');
            const target = targetUuidMap[uuid] || Object.values(targetUuidMap).find(t => t.uuid.includes(uuid.replace('trg-', '')));

            if (target) {
                context += `\n[CRITICAL TARGET DATA FOUND for ${uuid}: ${JSON.stringify(target)}]`;
                console.log("AI Context Found for Target:", uuid);
            }
        }

        try {
            const response = await fetch('http://localhost:8000/chat', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    message: text,
                    context: context
                })
            });
            const data = await response.json();
            messages.removeChild(typing);
            addMessage(data.response || "I couldn't generate a response.");
        } catch (e) {
            if (messages.contains(typing)) messages.removeChild(typing);
            addMessage("Error: AI Assistant API is offline. Check if web_api.py is running on port 8000.");
        }
    };

    send.onclick = handleSend;
    input.onkeypress = (e) => { if (e.key === 'Enter') handleSend(); };
})();
