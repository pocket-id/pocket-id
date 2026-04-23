/* Simple login page – no framework, no bundler needed.
   Depends on QRCode global from qrcode.min.js. */
(function () {
    'use strict';

    var SELF_LOGIN_CLIENT_ID = 'pocket-id-self-login';
    var SELF_LOGIN_SCOPE = 'openid profile email';
    var API_BASE = '/api/oidc/device';

    var deviceCode = '';
    var pollingInterval = 5;
    var expiresAt = 0;
    var pollTimer = null;
    var countdownTimer = null;
    var redirectUrl = '';

    function isSafeRedirect(url) {
        return url && url.charAt(0) === '/' && url.charAt(1) !== '/' && url.charAt(1) !== '\\';
    }

    function getRedirectParam() {
        var search = window.location.search;
        if (!search) return '';
        var params = search.substring(1).split('&');
        for (var i = 0; i < params.length; i++) {
            var pair = params[i].split('=');
            if (pair[0] === 'redirect') {
                var url = decodeURIComponent(pair[1] || '');
                return isSafeRedirect(url) ? url : '';
            }
        }
        return '';
    }

    function show(id) {
        var states = document.querySelectorAll('.state');
        for (var i = 0; i < states.length; i++) {
            states[i].className = 'state hidden';
        }
        var el = document.getElementById(id);
        el.className = 'state';
        el.focus();
    }

    function setText(id, text) {
        var el = document.getElementById(id);
        if (el) el.textContent = text;
    }

    function request(method, url, data, callback, retries) {
        if (retries === undefined) retries = 1;
        var xhr = new XMLHttpRequest();
        xhr.open(method, url, true);
        xhr.timeout = 15000;
        if (method === 'POST') {
            xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
        }

        function onError() {
            if (retries > 0) {
                setTimeout(function () {
                    request(method, url, data, callback, retries - 1);
                }, 2000);
            } else {
                callback(0, null);
            }
        }

        xhr.ontimeout = onError;
        xhr.onerror = onError;

        xhr.onreadystatechange = function () {
            if (xhr.readyState !== 4) return;
            if (xhr.status === 0) return; // handled by ontimeout/onerror
            var response = null;
            try {
                response = JSON.parse(xhr.responseText);
            } catch (e) {
                response = null;
            }
            callback(xhr.status, response);
        };

        xhr.send(data || null);
    }

    function formatTime(seconds) {
        var m = Math.floor(seconds / 60);
        var s = seconds % 60;
        return (m < 10 ? '0' : '') + m + ':' + (s < 10 ? '0' : '') + s;
    }

    function renderUserCode(code) {
        var container = document.getElementById('user-code');
        if (!container) return;
        container.textContent = '';
        var formatted = code.length > 1 ? code.substring(0, Math.floor(code.length / 2)) + ' \u2013 ' + code.substring(Math.floor(code.length / 2)) : code;
        container.setAttribute('aria-label', formatted);
        var half = Math.floor(code.length / 2);
        for (var i = 0; i < code.length; i++) {
            if (i === half && code.length > 1) {
                var sep = document.createElement('span');
                sep.className = 'code-separator';
                sep.setAttribute('aria-hidden', 'true');
                sep.textContent = '\u2013';
                container.appendChild(sep);
            }
            var box = document.createElement('span');
            box.className = 'code-box';
            box.setAttribute('aria-hidden', 'true');
            box.textContent = code.charAt(i);
            container.appendChild(box);
        }
    }

    function updateCountdown() {
        var remaining = Math.max(0, Math.floor((expiresAt - Date.now()) / 1000));
        setText('countdown', 'Code expires in ' + formatTime(remaining));
        if (remaining <= 0) {
            cleanup();
            show('expired-state');
        }
    }

    function cleanup() {
        if (pollTimer) {
            clearTimeout(pollTimer);
            pollTimer = null;
        }
        if (countdownTimer) {
            clearInterval(countdownTimer);
            countdownTimer = null;
        }
    }

    function renderQR(url) {
        var canvas = document.getElementById('qr-canvas');
        if (!canvas || !url) return;
        if (typeof QRCode === 'undefined') {
            console.error('QRCode library not loaded');
            return;
        }
        QRCode.toCanvas(canvas, url, { width: 200, margin: 0 }, function (err) {
            if (err) console.error('QR render failed:', err);
        });
    }

    function poll() {
        var params = 'device_code=' + encodeURIComponent(deviceCode) +
            '&client_id=' + encodeURIComponent(SELF_LOGIN_CLIENT_ID);

        request('POST', API_BASE + '/exchange-session', params, function (status, data) {
            if (status >= 200 && status < 300) {
                cleanup();
                show('authorized-state');
                setTimeout(function () {
                    window.location.href = redirectUrl || '/';
                }, 1000);
                return;
            }

            if (data && data.error) {
                if (data.error === 'authorization_pending') {
                    pollTimer = setTimeout(poll, pollingInterval * 1000);
                    return;
                }
                if (data.error === 'slow_down') {
                    pollingInterval = Math.max(pollingInterval + 5, data.interval || (pollingInterval + 5));
                    pollTimer = setTimeout(poll, pollingInterval * 1000);
                    return;
                }
                if (data.error === 'expired_token') {
                    cleanup();
                    show('expired-state');
                    return;
                }
                if (data.error === 'access_denied') {
                    cleanup();
                    setText('error-message', 'Login was denied on the other device.');
                    show('error-state');
                    return;
                }
                if (data.error === 'invalid_grant') {
                    cleanup();
                    setText('error-message', 'Invalid or unknown device code. Please try again.');
                    show('error-state');
                    return;
                }
            }

            if (status === 0 && !data) {
                // Network error — retry next poll cycle instead of giving up
                pollTimer = setTimeout(poll, pollingInterval * 1000);
                return;
            }

            cleanup();
            if (data && data.error) {
                console.error('Device flow error:', data.error);
            }
            setText('error-message', 'An unexpected error occurred. Please try again.');
            show('error-state');
        });
    }

    function startFlow() {
        cleanup();
        show('loading-state');

        var params = 'client_id=' + encodeURIComponent(SELF_LOGIN_CLIENT_ID) +
            '&scope=' + encodeURIComponent(SELF_LOGIN_SCOPE);

        request('POST', API_BASE + '/authorize', params, function (status, data) {
            if (status < 200 || status >= 300 || !data) {
                if (data && data.error) console.error('Device authorize error:', data.error);
                setText('error-message', 'Failed to start the login. Please try again.');
                show('error-state');
                return;
            }

            if (!data.device_code || !data.user_code || !data.verification_uri_complete) {
                setText('error-message', 'Invalid response from server.');
                show('error-state');
                return;
            }

            deviceCode = data.device_code;
            pollingInterval = data.interval || 5;
            expiresAt = Date.now() + (data.expires_in || 300) * 1000;

            renderQR(data.verification_uri_complete);
            renderUserCode(data.user_code || '');

            updateCountdown();
            countdownTimer = setInterval(updateCountdown, 1000);

            show('showing-state');
            pollTimer = setTimeout(poll, pollingInterval * 1000);
        });
    }

    function loadAppName() {
        request('GET', '/api/application-configuration', null, function (status, data) {
            if (status >= 200 && status < 300 && data) {
                for (var i = 0; i < data.length; i++) {
                    if (data[i].key === 'appName' && data[i].value) {
                        setText('app-name', data[i].value);
                        document.title = 'Sign In - ' + data[i].value;
                        return;
                    }
                }
            }
        });
    }

    redirectUrl = getRedirectParam();
    loadAppName();
    startFlow();

    document.getElementById('expired-btn').addEventListener('click', startFlow);
    document.getElementById('retry-btn').addEventListener('click', startFlow);
})();
