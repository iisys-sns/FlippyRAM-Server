function redirectHash() {
    var hash = document.getElementById('hash').value;
    console.log(hash);
    if (hash.match(/^[0-9a-f]{64}$/)) {
        window.location.href = '/token/' + hash;
    } else {
        window.location.href = '/error.html';
    }
}

function hammertime() {
    const hammertime = document.getElementById('js-hammertime');
    let clickTimeout;

    hammertime.addEventListener('click', function (event) {
        event.preventDefault();
        if (!clickTimeout) {
            clickTimeout = setTimeout(() => {
                clickTimeout = null;
                window.location.href = event.target.href;
            }, 500);
        }
    });

    hammertime.addEventListener('dblclick', function (event) {
        const child = this.firstElementChild;
        clearTimeout(clickTimeout);
        clickTimeout = null;
        if (child) {
            child.classList.toggle('hammertime');
        }
    });
}

window.onload = function () {
    if (document.getElementById('hashForm') != null) {
        document.getElementById('hashForm').onsubmit = function () {
            event.preventDefault();
            redirectHash();
        };
    }

    hammertime();
};
