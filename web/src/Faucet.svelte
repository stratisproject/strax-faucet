<script>
  import { onMount } from 'svelte';
  import { getAddress } from '@ethersproject/address';
  import { CloudflareProvider } from '@ethersproject/providers';
  import { setDefaults as setToast, toast } from 'bulma-toast';
  import SocialIcons from '@rodneylab/svelte-social-icons';

  let input = null;
  let faucetInfo = {
    account: '0x0000000000000000000000000000000000000000',
    network: 'testnet',
    payout: 1,
    symbol: 'ETH',
    hcaptcha_sitekey: '',
    discord_client_id: '',
  };

  let loggedIn = false;
  let mounted = false;
  let hcaptchaLoaded = false;
  let loginUrl = '';

  onMount(async () => {

    const params = new URLSearchParams(window.location.search);
    const code = params.get('code');

    const res = await fetch('/api/info');
    faucetInfo = await res.json();
    loginUrl = `https://discord.com/api/oauth2/authorize?client_id=${faucetInfo.discord_client_id}&redirect_uri=${window.location.href}&response_type=code&scope=identify%20email`;

    await checkAuthentication();

    if (code) {
      exchangeCodeForToken(code);      
    }

    await checkNetwork();
    if (window.ethereum) {
        window.ethereum.on('chainChanged', (_chainId) => {
          checkNetwork();
        });
    }
    
    mounted = true;    
  });

  async function checkAuthentication() {
    try {
      const response = await fetch('/api/check', {
        credentials: 'include' // Ensures cookies are included in the request
      });

      if (response.ok) {
        loggedIn = true;
      } else {
        loggedIn = false;
      }
    } catch (error) {
      console.error('Error checking authentication:', error);
    }
  }

  async function exchangeCodeForToken(code) {
    // Make an API request to your backend with the code
    const response = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    });

    if (response.ok) {
      loggedIn = true;
      const newUrl = window.location.pathname; // This retains the current path without the query parameters
      window.history.replaceState({}, '', newUrl);
    } else {
      // Handle errors
    }
  }

  window.hcaptchaOnLoad = () => {
    hcaptchaLoaded = true;
  };

  $: document.title = `${faucetInfo.symbol} ${capitalize(
    faucetInfo.network,
  )} Faucet`;

  let widgetID;
  $: if (mounted && hcaptchaLoaded) {
    widgetID = window.hcaptcha.render('hcaptcha', {
      sitekey: faucetInfo.hcaptcha_sitekey,
    });
  }

  let accounts = [];

  async function connectMetaMask() {
    if (window.ethereum) {
      try {
        // Request account access
        accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
        console.log('Connected account:', accounts[0]);
        input = accounts[0];
        // Proceed with the connected account
      } catch (error) {
        if (error.code === 4001) {
          // User denied account access
          console.log('User denied account access');
        } else {
          console.error('Error connecting to MetaMask:', error);
        }
      }
    } else {
      console.log('MetaMask is not installed');
    }
  }

  let networkLabel = 'Checking Network...';
  
  async function checkNetwork() {
    setTimeout(async () => {
      if (window.ethereum) {
        try {
          const currentChainId = window.ethereum.chainId;

          if (currentChainId === '0x32195') {
            networkLabel = 'Straits Auroria Testnet'; // Correct network
            await connectMetaMask();
          } else {
            networkLabel = 'Switch Network'; // Incorrect network
          }
        } catch (error) {
          console.error('Error checking the network:', error);
          networkLabel = 'Error';
        }
      } else {
          console.log('Ethereum wallet not detected');
          networkLabel = 'No Wallet Detected';
      }
    }, 2000);
  }

  async function addCustomNetwork() {
    if (window.ethereum) {
      try {
        // Request to add a custom network
        await window.ethereum.request({
          method: 'wallet_addEthereumChain',
          params: [{
            chainId: '0x32195', // The chainId of the network in hexadecimal, 205205 in decimal
            chainName: 'Straits Auroria Testnet',
            nativeCurrency: {
                name: 'STRAX',
                symbol: 'STRAX', // Up to 5 characters
                decimals: 18
            },
            rpcUrls: ['https://auroria.rpc.stratisevm.com/'],
            blockExplorerUrls: ['https://auroria.explorer.stratisevm.com/']
          }],
        });
      } catch (addError) {
        alert('Error adding Auroria network:');
      }
    } else {
      alert('Metamask wallet is not installed');
    }
}

  setToast({
    position: 'bottom-center',
    dismissible: true,
    pauseOnHover: true,
    closeOnClick: false,
    animate: { in: 'fadeIn', out: 'fadeOut' },
  });

  async function handleRequest() {
    let address = input;
    if (address === null) {
      toast({ message: 'input required', type: 'is-warning' });
      return;
    }

    if (address.endsWith('.eth')) {
      try {
        const provider = new CloudflareProvider();
        address = await provider.resolveName(address);
        if (!address) {
          toast({ message: 'invalid ENS name', type: 'is-warning' });
          return;
        }
      } catch (error) {
        toast({ message: error.reason, type: 'is-warning' });
        return;
      }
    }

    try {
      address = getAddress(address);
    } catch (error) {
      toast({ message: error.reason, type: 'is-warning' });
      return;
    }

    try {
      let headers = {
        'Content-Type': 'application/json',
      };

      if (hcaptchaLoaded) {
        const { response } = await window.hcaptcha.execute(widgetID, {
          async: true,
        });
        headers['h-captcha-response'] = response;
      }

      const res = await fetch('/api/claim', {
        method: 'POST',
        credentials: 'include',
        headers,
        body: JSON.stringify({
          address,
        }),
      });

      let { msg } = await res.json();
      let type = res.ok ? 'is-success' : 'is-warning';
      toast({ message: msg, type });
    } catch (err) {
      console.error(err);
    }
  }

  function capitalize(str) {
    const lower = str.toLowerCase();
    return str.charAt(0).toUpperCase() + lower.slice(1);
  }

  const onload = el => {
    // Init ParticleAnimation
    const canvasElements = document.querySelectorAll('[data-particle-animation]');
    canvasElements.forEach(canvas => {
      const options = {
        quantity: canvas.dataset.particleQuantity,
        staticity: canvas.dataset.particleStaticity,
        ease: canvas.dataset.particleEase,
      };
      new ParticleAnimation(canvas, options);
    });
  }

  AOS.init({
    once: true,
    disable: 'phone',
    duration: 1000,
    easing: 'ease-out-cubic',
  });

  // Particle animation
  class ParticleAnimation {
    constructor(el, { quantity = 30, staticity = 50, ease = 50 } = {}) {
      this.canvas = el;
      if (!this.canvas) return;
      this.canvasContainer = this.canvas.parentElement;
      this.context = this.canvas.getContext('2d');
      this.dpr = window.devicePixelRatio || 1;
      this.settings = {
        quantity: quantity,
        staticity: staticity,
        ease: ease,
      };
      this.circles = [];
      this.mouse = {
        x: 0,
        y: 0,
      };
      this.canvasSize = {
        w: 0,
        h: 0,
      };
      this.onMouseMove = this.onMouseMove.bind(this);
      this.initCanvas = this.initCanvas.bind(this);
      this.resizeCanvas = this.resizeCanvas.bind(this);
      this.drawCircle = this.drawCircle.bind(this);
      this.drawParticles = this.drawParticles.bind(this);
      this.remapValue = this.remapValue.bind(this);
      this.animate = this.animate.bind(this);
      this.init();
    }

    init() {
      this.initCanvas();
      this.animate();
      window.addEventListener('resize', this.initCanvas);
      window.addEventListener('mousemove', this.onMouseMove);
    }

    initCanvas() {
      this.resizeCanvas();
      this.drawParticles();
    }

    onMouseMove(event) {
      const { clientX, clientY } = event;
      const rect = this.canvas.getBoundingClientRect();
      const { w, h } = this.canvasSize;
      const x = clientX - rect.left - (w / 2);
      const y = clientY - rect.top - (h / 2);
      const inside = x < (w / 2) && x > -(w / 2) && y < (h / 2) && y > -(h / 2);
      if(inside) {
        this.mouse.x = x;
        this.mouse.y = y;
      }
    }

    resizeCanvas() {
      this.circles.length = 0;
      this.canvasSize.w = this.canvasContainer.offsetWidth;
      this.canvasSize.h = this.canvasContainer.offsetHeight;
      this.canvas.width = this.canvasSize.w * this.dpr;
      this.canvas.height = this.canvasSize.h * this.dpr;
      this.canvas.style.width = this.canvasSize.w + 'px';
      this.canvas.style.height = this.canvasSize.h + 'px';
      this.context.scale(this.dpr, this.dpr);
    }

    circleParams() {
      const x = Math.floor(Math.random() * this.canvasSize.w);
      const y = Math.floor(Math.random() * this.canvasSize.h);
      const translateX = 0;
      const translateY = 0;
      const size = Math.floor(Math.random() * 2) + 1;
      const alpha = 0;
      const targetAlpha = parseFloat((Math.random() * 0.6 + 0.1).toFixed(1));
      const dx = (Math.random() - 0.5) * 0.2;
      const dy = (Math.random() - 0.5) * 0.2;
      const magnetism = 0.1 + Math.random() * 4;
      return { x, y, translateX, translateY, size, alpha, targetAlpha, dx, dy, magnetism };
    }

    drawCircle(circle, update = false) {
      const { x, y, translateX, translateY, size, alpha } = circle;
      this.context.translate(translateX, translateY);
      this.context.beginPath();
      this.context.arc(x, y, size, 0, 2 * Math.PI);
      this.context.fillStyle = `rgba(255, 255, 255, ${alpha})`;
      this.context.fill();
      this.context.setTransform(this.dpr, 0, 0, this.dpr, 0, 0);
      if (!update) {
        this.circles.push(circle);
      }
    }

    clearContext() {
      this.context.clearRect(0, 0, this.canvasSize.w, this.canvasSize.h);
    }  

    drawParticles() {
      this.clearContext();
      const particleCount = this.settings.quantity;
      for (let i = 0; i < particleCount; i++) {
        const circle = this.circleParams();
        this.drawCircle(circle);
      }
    }

    // This function remaps a value from one range to another range
    remapValue(value, start1, end1, start2, end2) {
      const remapped = (value - start1) * (end2 - start2) / (end1 - start1) + start2;
      return remapped > 0 ? remapped : 0;
    }

    animate() {
      this.clearContext();
      this.circles.forEach((circle, i) => {
        // Handle the alpha value
        const edge = [
          circle.x + circle.translateX - circle.size, // distance from left edge
          this.canvasSize.w - circle.x - circle.translateX - circle.size, // distance from right edge
          circle.y + circle.translateY - circle.size, // distance from top edge
          this.canvasSize.h - circle.y - circle.translateY - circle.size, // distance from bottom edge
        ];
        const closestEdge = edge.reduce((a, b) => Math.min(a, b));
        const remapClosestEdge = this.remapValue(closestEdge, 0, 20, 0, 1).toFixed(2);
        if(remapClosestEdge > 1) {
          circle.alpha += 0.02;
          if(circle.alpha > circle.targetAlpha) circle.alpha = circle.targetAlpha;
        } else {
          circle.alpha = circle.targetAlpha * remapClosestEdge;
        }
        circle.x += circle.dx;
        circle.y += circle.dy;
        circle.translateX += ((this.mouse.x / (this.settings.staticity / circle.magnetism)) - circle.translateX) / this.settings.ease;
        circle.translateY += ((this.mouse.y / (this.settings.staticity / circle.magnetism)) - circle.translateY) / this.settings.ease;
        // circle gets out of the canvas
        if (circle.x < -circle.size || circle.x > this.canvasSize.w + circle.size || circle.y < -circle.size || circle.y > this.canvasSize.h + circle.size) {
          // remove the circle from the array
          this.circles.splice(i, 1);
          // create a new circle
          const circle = this.circleParams();
          this.drawCircle(circle);
          // update the circle position
        } else {
          this.drawCircle({ ...circle, x: circle.x, y: circle.y, translateX: circle.translateX, translateY: circle.translateY, alpha: circle.alpha }, true);
        }
      });
      window.requestAnimationFrame(this.animate);
    }
  }
</script>

<svelte:head>
  {#if mounted && faucetInfo.hcaptcha_sitekey}
    <script
      src="https://hcaptcha.com/1/api.js?onload=hcaptchaOnLoad&render=explicit"
      async
      defer
    ></script>
  {/if}
</svelte:head>

<!-- Site header -->
<header class="absolute w-full z-30">
  <div class="max-w-6xl mx-auto px-4 sm:px-6">
    <div class="flex items-center justify-between h-16 md:h-20">

      <!-- Site branding -->
      <div class="flex-1">
        <!-- Logo -->
        <a class="inline-flex items-center" href="index.html" aria-label="Cruip">
          <img class="max-w-none" src="/images/stratis_logo_white.svg" width="38" height="38" alt="Stellar">
          <span class="ml-3 hidden md:block">Stratis Auroria Faucet</span>
        </a>
      </div>

      <!-- Desktop sign in links -->
      <ul class="flex-1 flex justify-end items-center">
        <li class="ml-6">
          <button class="btn-sm text-slate-300 hover:text-white transition duration-150 ease-in-out w-full group [background:linear-gradient(theme(colors.slate.900),_theme(colors.slate.900))_padding-box,_conic-gradient(theme(colors.slate.400),_theme(colors.slate.700)_25%,_theme(colors.slate.700)_75%,_theme(colors.slate.400)_100%)_border-box] relative before:absolute before:inset-0 before:bg-slate-800/30 before:rounded-full before:pointer-events-none"
          on:click={addCustomNetwork}>
            <span class="relative inline-flex items-center">
              {networkLabel} <span
                class="tracking-normal text-purple-500 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">-&gt;</span>
            </span>
          </button>
        </li>
      </ul>
    </div>
  </div>
</header>

<!-- Page content -->
<main class="grow" use:onload>
  <!-- Features #2 -->
  <section class="relative">

    <!-- Particles animation -->
    <div class="absolute left-1/2 -translate-x-1/2 top-0 -z-10 w-80 h-80 -mt-24 -ml-32">
      <div class="absolute inset-0 -z-10" aria-hidden="true">
        <canvas data-particle-animation data-particle-quantity="6" data-particle-staticity="30"></canvas>
      </div>
    </div>

    <div class="max-w-6xl mx-auto px-4 sm:px-6">
      <div class="pt-16 pt-32">

      <!-- Particles animation -->
      <div class="absolute inset-0 -z-10" aria-hidden="true">
        <canvas data-particle-animation></canvas>
      </div>

      <!-- Illustration -->
      <div class="absolute inset-0 -z-10 -mx-28 rounded-b-[3rem] pointer-events-none overflow-hidden"
        aria-hidden="true">
        <div class="absolute left-1/2 -translate-x-1/2 bottom-0 -z-10">
          <img src="/images/glow-bottom.svg" class="max-w-none" width="2146" height="774" alt="Hero Illustration">
        </div>
      </div>

        <!-- Section header -->
        <div class="max-w-xl mx-auto text-center md:px-1 mt-20 px-5 pb-20 md:pb-20">
          <h2
            class="h2 bg-clip-text text-transparent bg-gradient-to-r from-slate-200/60 via-slate-200 to-slate-200/60 pb-4">
            Receive {faucetInfo.payout} {faucetInfo.symbol} per request.</h2>
          <p class="md:text-lg text-sm text-slate-400">Serving from {faucetInfo.account}</p>
          <div id="hcaptcha" data-size="invisible"></div>
          <div class="py-12 px-12 w-xl mx-auto">
            <div class="mb-40">
              <div class="space-y-2">
                <div>
                  <label class="block text-sm text-slate-300 font-medium mb-1" for="address">tSTRAX Address</label>
                  <input id="address" class="form-input w-full h-3 p-3 rounded-full" type="text" bind:value={input} required placeholder="Enter your address" />
                </div>
              </div>
              <div class="mt-2">
                {#if loggedIn}
                  <button on:click={handleRequest} type="button"
                    class="btn text-slate-300 hover:text-white transition duration-150 ease-in-out w-full group [background:linear-gradient(theme(colors.slate.900),_theme(colors.slate.900))_padding-box,_conic-gradient(theme(colors.slate.400),_theme(colors.slate.700)_25%,_theme(colors.slate.700)_75%,_theme(colors.slate.400)_100%)_border-box] relative before:absolute before:inset-0 before:bg-slate-800/30 before:rounded-full before:pointer-events-none">
                    Request <span
                      class="tracking-normal text-purple-300 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">-&gt;</span>
                  </button>                
                {:else}
                  <a href={loginUrl}
                    class="btn text-slate-300 hover:text-white transition duration-150 ease-in-out w-full group [background:linear-gradient(theme(colors.slate.900),_theme(colors.slate.900))_padding-box,_conic-gradient(theme(colors.slate.400),_theme(colors.slate.700)_25%,_theme(colors.slate.700)_75%,_theme(colors.slate.400)_100%)_border-box] relative before:absolute before:inset-0 before:bg-slate-800/30 before:rounded-full before:pointer-events-none">
                    Login with Discord <span
                      class="tracking-normal text-purple-300 group-hover:translate-x-0.5 transition-transform duration-150 ease-in-out ml-1">-&gt;</span>
                  </a>
                {/if}
              </div>
              <p>&nbsp;</p>
              <p>&nbsp;</p>
              <p>&nbsp;</p>
              <p>&nbsp;</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</main>

<!-- Site footer -->
<footer>
  <div class="max-w-6xl mx-auto px-4 sm:px-6">

    <!-- Blocks -->
    <div class="grid sm:grid-cols-12 gap-8 py-8 md:py-4">

      <!-- 1st block -->
      <div class="sm:col-span-12 lg:col-span-4 order-1 lg:order-none">
        <div class="h-full flex flex-col sm:flex-row lg:flex-col justify-between">
          <div class="mb-4">
            <div class="mb-4">
              <!-- Logo -->
              <a class="inline-flex" href="index.html" aria-label="Cruip">
                <img src="/images/stratis_logo_white.svg" width="38" height="38" alt="Stellar">
              </a>
            </div>
            <div class="text-sm text-slate-300">&copy; Stratis Platform <span class="text-slate-500">-</span> All
              rights
              reserved.</div>
          </div>
          <!-- Social links -->
          <ul class="flex mt-3">
            <li>
              <a class="flex justify-center items-center text-purple-500 hover:text-purple-400 transition duration-150 ease-in-out"
                href="https://twitter.com/stratisplatform" aria-label="Twitter">
                <SocialIcons width="32" height="32" fgColor="#a855f7" bgColor="#262626" network="twitter" alt="twitter"/>
              </a>
            </li>
            <li class="ml-2">
              <a class="flex justify-center items-center text-purple-500 hover:text-purple-400 transition duration-150 ease-in-out"
                href="https://github.com/stratisproject" aria-label="Dev.to">
                <SocialIcons width="32" height="32" fgColor="#a855f7" bgColor="#262626" network="discord" alt="discord"/>
              </a>
            </li>
            <li class="ml-2">
              <a class="flex justify-center items-center text-purple-500 hover:text-purple-400 transition duration-150 ease-in-out"
                href="https://discordapp.com/invite/9tDyfZs" aria-label="Github">
                <SocialIcons width="32" height="32" fgColor="#a855f7" bgColor="#262626" network="github" alt="github"/>
              </a>
            </li>
          </ul>
        </div>
      </div>

    </div>

  </div>
</footer>

