<template>
  <section class="legal-page">
    <h2>Terms of Service</h2>
    <p class="effective">Effective date: August 25, 2026</p>

    <h3>1. Acceptance of Terms</h3>
    <p>
      By using Hyperfocus ("the Software"), you agree to be bound by these Terms
      of Service. If you do not agree, do not use the Software.
    </p>

    <h3>2. Description of Service</h3>
    <p>
      Hyperfocus polls the Twitch Helix API to track live broadcasters of Dead
      by Daylight, records stream metadata and preview thumbnails, and makes
      that history browsable on this website. An optional notification bot
      matches your Steam name against the players spotted in other streamers'
      lobbies and pings you in Twitch chat when it finds a match.
    </p>

    <h3>3. User Responsibilities</h3>
    <p>
      By subscribing to the notification bot you confirm that the Steam profile
      you submit is your own, and you agree that the bot may contact you in
      Twitch chat. You may unsubscribe at any time via the chat command, which
      deletes your subscription data. When self-hosting the Software you must
      comply with Twitch's Developer Terms of Service and API usage policies
      and are responsible for securing your own instance.
    </p>

    <h3>4. Intellectual Property</h3>
    <p>
      The Software is open source and distributed under its included license.
      Dead by Daylight and all related trademarks, logos, and game content are
      the property of Behaviour Interactive. Twitch-related trademarks, logos,
      and content are the property of Twitch Interactive. Steam-related
      trademarks and content are the property of Valve Corporation. Hyperfocus
      is an independent fan project, is not affiliated with or endorsed by any
      of those companies, and respects their intellectual property rights.
      Preview thumbnails and stream metadata are retrieved from Twitch's public
      API and are used in accordance with Twitch's policies.
    </p>

    <h3>5. Disclaimer of Warranties</h3>
    <p>
      THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
      IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
      FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
      AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
      LIABILITY ARISING FROM THE USE OF THE SOFTWARE.
    </p>

    <h3>6. Limitation of Liability</h3>
    <p>
      Under no circumstances shall the authors or copyright holders be liable for
      any direct, indirect, incidental, special, exemplary, or consequential
      damages arising from the use of or inability to use the Software, even if
      advised of the possibility of such damages. This includes, but is not
      limited to, data loss, service interruptions, or issues arising from Twitch
      API changes or rate limiting.
    </p>

    <h3>7. Changes to Terms</h3>
    <p>
      These terms may be updated at any time. Continued use of the Software after
      changes constitutes acceptance of the revised terms.
    </p>

    <h3>8. Governing Law</h3>
    <p>
      These terms are governed by the laws applicable to the jurisdiction in which
      the Software's authors reside, without regard to conflict of law principles.
    </p>
  </section>
</template>

<style scoped>
.legal-page {
  max-width: 640px;
  margin: 0 auto;
}

.legal-page h2 {
  margin-top: 0;
  font-size: 1.1rem;
}

.legal-page h3 {
  font-size: 0.95rem;
  margin-top: 1.5rem;
  margin-bottom: 0.25rem;
}

.legal-page p {
  line-height: 1.6;
  font-size: 0.9rem;
  color: var(--p-text-muted-color, #94a3b8);
  margin: 0.25rem 0 0.5rem;
}

.legal-page .effective {
  font-size: 0.8rem;
  margin-bottom: 1rem;
}
</style>
