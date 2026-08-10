<template>
  <section class="legal-page">
    <h2>Terms of Service</h2>
    <p class="effective">Effective date: {{ new Date().getFullYear() }}</p>

    <h3>1. Acceptance of Terms</h3>
    <p>
      By using Hyperfocus ("the Software"), you agree to be bound by these Terms
      of Service. If you do not agree, do not use the Software.
    </p>

    <h3>2. Description of Service</h3>
    <p>
      Hyperfocus is a self-hosted application that polls the Twitch Helix API to
      track live broadcasters for a configured game category. It records stream
      metadata and preview thumbnails locally. It does not provide hosted services
      or process data on behalf of users.
    </p>

    <h3>3. User Responsibilities</h3>
    <p>
      You are responsible for deploying and maintaining the Software on your own
      infrastructure. You must comply with Twitch's Developer Terms of Service and
      API usage policies when configuring and operating the Software. You are
      responsible for securing your own instance, including the PostgreSQL
      database and any network exposure.
    </p>

    <h3>4. Intellectual Property</h3>
    <p>
      The Software is open source and distributed under its included license.
      Twitch-related trademarks, logos, and content remain the property of Twitch
      Interactive, Inc. Preview thumbnails and stream metadata are retrieved from
      Twitch's public API and are subject to Twitch's intellectual property
      policies.
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
