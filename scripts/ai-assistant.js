(function () {
    // ── Duplicate guard ──────────────────────────────────────────
    if (document.getElementById('ai-chat-widget')) return;

    // ── Styles ───────────────────────────────────────────────────
    const style = document.createElement('style');
    style.textContent = [
        '#ai-chat-widget{position:fixed;bottom:24px;right:24px;z-index:99999;font-family:Inter,sans-serif}',
        '#ai-chat-toggle{width:58px;height:58px;border-radius:50%;background:linear-gradient(135deg,#8b5cf6,#6d28d9);color:#fff;border:none;box-shadow:0 4px 20px rgba(139,92,246,.5);cursor:pointer;display:flex;align-items:center;justify-content:center;transition:transform .3s,box-shadow .3s;position:relative}',
        '#ai-chat-toggle:hover{transform:scale(1.08);box-shadow:0 6px 28px rgba(139,92,246,.7)}',
        '#ai-chat-toggle .ai-pulse{position:absolute;top:4px;right:4px;width:12px;height:12px;border-radius:50%;background:#10b981;border:2px solid #0f1115;animation:ai-pulse-anim 2s infinite}',
        '@keyframes ai-pulse-anim{0%,100%{transform:scale(1);opacity:1}50%{transform:scale(1.3);opacity:.7}}',
        '#ai-chat-window{display:none;position:absolute;bottom:72px;right:0;width:370px;height:520px;background:rgba(22,25,35,.97);border:1px solid rgba(139,92,246,.3);border-radius:16px;box-shadow:0 20px 60px rgba(0,0,0,.6);flex-direction:column;overflow:hidden;backdrop-filter:blur(20px);color:#fff}',
        '.ai-hdr{padding:14px 16px;background:linear-gradient(135deg,#8b5cf6,#6d28d9);font-weight:700;font-size:.9rem;display:flex;justify-content:space-between;align-items:center;flex-shrink:0}',
        '.ai-hdr-left{display:flex;align-items:center;gap:10px}',
        '.ai-status-dot{width:8px;height:8px;border-radius:50%;background:#10b981;display:inline-block}',
        '.ai-msgs{flex:1;overflow-y:auto;padding:14px;display:flex;flex-direction:column;gap:10px;scrollbar-width:thin;scrollbar-color:rgba(139,92,246,.3) transparent}',
        '.ai-msgs::-webkit-scrollbar{width:4px}',
        '.ai-msgs::-webkit-scrollbar-thumb{background:rgba(139,92,246,.4);border-radius:2px}',
        '.ai-quick-actions{padding:0 14px 10px;display:flex;gap:6px;flex-wrap:wrap;flex-shrink:0}',
        '.ai-quick-btn{font-size:.7rem;padding:4px 10px;border-radius:20px;background:rgba(139,92,246,.15);color:#c4b5fd;border:1px solid rgba(139,92,246,.3);cursor:pointer;transition:background .2s}',
        '.ai-quick-btn:hover{background:rgba(139,92,246,.3)}',
        '.ai-inp-area{padding:12px 14px;border-top:1px solid rgba(255,255,255,.06);background:rgba(0,0,0,.2);display:flex;gap:8px;align-items:flex-end;flex-shrink:0}',
        '.ai-inp{flex:1;background:rgba(255,255,255,.05);border:1px solid rgba(255,255,255,.1);border-radius:10px;padding:9px 12px;color:#fff;font-size:.82rem;outline:none;resize:none;height:38px;max-height:90px;transition:border-color .2s;font-family:inherit}',
        '.ai-inp:focus{border-color:rgba(139,92,246,.5)}',
        '.ai-snd{background:#8b5cf6;color:#fff;border:none;border-radius:10px;width:36px;height:36px;cursor:pointer;display:flex;align-items:center;justify-content:center;flex-shrink:0;transition:background .2s}',
        '.ai-snd:hover{background:#7c3aed}',
        '.msg-user{align-self:flex-end;background:linear-gradient(135deg,#8b5cf6,#7c3aed);color:#fff;padding:9px 13px;border-radius:14px 14px 2px 14px;font-size:.82rem;max-width:82%;line-height:1.45}',
        '.msg-ai{align-self:flex-start;background:rgba(255,255,255,.06);color:#e2e8f0;padding:9px 13px;border-radius:14px 14px 14px 2px;font-size:.82rem;max-width:85%;line-height:1.5;border:1px solid rgba(255,255,255,.07)}',
        '.ai-typing{display:flex;align-items:center;gap:8px;color:#94a3b8;font-size:.78rem;padding:4px 0}',
        '.ai-typing-dots span{display:inline-block;width:5px;height:5px;border-radius:50%;background:#8b5cf6;margin:0 1px;animation:ai-dot 1.4s infinite both}',
        '.ai-typing-dots span:nth-child(2){animation-delay:.2s}',
        '.ai-typing-dots span:nth-child(3){animation-delay:.4s}',
        '@keyframes ai-dot{0%,80%,100%{transform:scale(0)}40%{transform:scale(1)}}'
    ].join('');
    document.head.appendChild(style);

    // ── Widget HTML ───────────────────────────────────────────────
    const widget = document.createElement('div');
    widget.id = 'ai-chat-widget';
    widget.innerHTML =
        '<button id="ai-chat-toggle" title="Observability AI Assistant">' +
            '<i class="fa-solid fa-robot" style="font-size:1.4rem"></i>' +
            '<span class="ai-pulse"></span>' +
        '</button>' +
        '<div id="ai-chat-window">' +
            '<div class="ai-hdr">' +
                '<div class="ai-hdr-left"><span class="ai-status-dot"></span><span>&#x1F985; Hawk-Eye AI</span></div>' +
                '<i class="fa-solid fa-xmark" id="ai-chat-close" style="cursor:pointer;opacity:.8"></i>' +
            '</div>' +
            '<div id="ai-chat-messages" class="ai-msgs">' +
                '<div class="msg-ai">Hi! I\'m your Hawk-Eye Observability AI.<br><br>' +
                'I can help you analyze system health, diagnose latency issues, decode alert patterns, or explain any metric.<br><br>' +
                'Ask anything \u2014 or pick a quick action below.</div>' +
            '</div>' +
            '<div class="ai-quick-actions">' +
                '<button class="ai-quick-btn" data-q="What targets are critical right now?">\uD83D\uDD34 Critical?</button>' +
                '<button class="ai-quick-btn" data-q="Why is DNS resolution failing?">\uD83C\uDF10 DNS?</button>' +
                '<button class="ai-quick-btn" data-q="Explain high latency on amazon.com">\u26A1 Latency?</button>' +
                '<button class="ai-quick-btn" data-q="Which SSL certs are expiring soon?">\uD83D\uDD12 SSL?</button>' +
            '</div>' +
            '<div class="ai-inp-area">' +
                '<textarea id="ai-chat-input" class="ai-inp" placeholder="Ask about system health, alerts, latency..." rows="1"></textarea>' +
                '<button id="ai-chat-send" class="ai-snd"><i class="fa-solid fa-paper-plane"></i></button>' +
            '</div>' +
        '</div>';
    document.body.appendChild(widget);

    // ── Logic ─────────────────────────────────────────────────────
    var aiWin  = document.getElementById('ai-chat-window');
    var toggle = document.getElementById('ai-chat-toggle');
    var close  = document.getElementById('ai-chat-close');
    var input  = document.getElementById('ai-chat-input');
    var send   = document.getElementById('ai-chat-send');
    var msgs   = document.getElementById('ai-chat-messages');

    toggle.onclick = function () {
        var open = aiWin.style.display === 'flex';
        aiWin.style.display = open ? 'none' : 'flex';
        toggle.style.transform = open ? 'scale(1)' : 'scale(0.92) rotate(15deg)';
    };
    close.onclick = function () {
        aiWin.style.display = 'none';
        toggle.style.transform = 'scale(1)';
    };

    document.querySelectorAll('.ai-quick-btn').forEach(function (btn) {
        btn.onclick = function () { input.value = btn.getAttribute('data-q'); handleSend(); };
    });

    function addMsg(html, isUser) {
        var div = document.createElement('div');
        div.className = isUser ? 'msg-user' : 'msg-ai';
        div.innerHTML = html;
        msgs.appendChild(div);
        msgs.scrollTop = msgs.scrollHeight;
        return div;
    }

    function handleSend() {
        var text = input.value.trim();
        if (!text) return;
        addMsg(text, true);
        input.value = '';

        // Disable send while waiting
        send.disabled = true;
        send.style.opacity = '0.5';

        var typing = document.createElement('div');
        typing.className = 'ai-typing';
        var typingLabel = document.createElement('span');
        typingLabel.textContent = 'Contacting AI\u2026';
        var dots = document.createElement('div');
        dots.className = 'ai-typing-dots';
        dots.innerHTML = '<span></span><span></span><span></span>';
        typing.appendChild(dots);
        typing.appendChild(typingLabel);
        msgs.appendChild(typing);
        msgs.scrollTop = msgs.scrollHeight;

        // Rotate status messages so it never looks frozen
        var statusMessages = ['Contacting AI\u2026', 'Sending query\u2026', 'Generating response\u2026', 'Almost there\u2026'];
        var statusIdx = 0;
        var statusInterval = setInterval(function () {
            statusIdx = (statusIdx + 1) % statusMessages.length;
            typingLabel.textContent = statusMessages[statusIdx];
        }, 4000);

        // Context enrichment
        var context = '';
        var incMatch = text.match(/inc-[a-z0-9]{4,12}/i);
        if (incMatch) {
            try {
                var uuidMap = JSON.parse(localStorage.getItem('hawkeye_uuid_map') || '{}');
                var inc = uuidMap[incMatch[0].toLowerCase()];
                if (inc) context = '\n[INCIDENT DATA: ' + JSON.stringify(inc) + ']';
            } catch (e) { /* ignore */ }
        }

        // Also inject current dashboard state as context
        var dashContext = '';
        try {
            if (typeof state !== 'undefined' && state.targets) {
                var problem = [];
                Object.values(state.targets).forEach(function(t) {
                    if (t.health && t.health.status && t.health.status !== 'healthy') {
                        problem.push('[CRITICAL: ' + t.name + ' is ' + t.health.status + ']');
                    }
                });
                if (problem.length > 0) {
                    dashContext = '\nCRITICAL SYSTEM CONTEXT:\n' + problem.join('\n');
                }
            }
        } catch (e) { /* ignore */ }

        var fullContext = (context + dashContext).trim();

        // Timeout: abort after 90s
        var controller = new AbortController();
        var timeout = setTimeout(function() { controller.abort(); }, 90000);

        fetch('http://localhost:8000/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message: text, context: fullContext }),
            signal: controller.signal
        }).then(function (res) {
            clearTimeout(timeout);
            clearInterval(statusInterval);
            return res.json();
        }).then(function (data) {
            if (msgs.contains(typing)) msgs.removeChild(typing);
            addMsg(data.response || "I couldn\u2019t generate a response.");
        }).catch(function (err) {
            clearTimeout(timeout);
            clearInterval(statusInterval);
            if (msgs.contains(typing)) msgs.removeChild(typing);
            if (err && err.name === 'AbortError') {
                addMsg('\u23F1 Request timed out — Ollama model may still be loading.<br><small style="color:#94a3b8">Try again in a moment. Model cold-start can take 30\u201360s.</small>');
            } else {
                addMsg('\u26A0\uFE0F AI Assistant is currently offline.<br><small style="color:#94a3b8">Start <code>web_api.py</code> on port 8000 to enable AI.</small>');
            }
        }).finally(function () {
            send.disabled = false;
            send.style.opacity = '1';
        });
    }

    send.onclick = handleSend;
    input.addEventListener('keypress', function (e) {
        if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(); }
    });
})();
