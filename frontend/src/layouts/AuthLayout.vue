<template>
  <div class="auth-page">
    <section class="auth-hero">
      <RouterLink class="brand brand-light" to="/"
        ><span class="brand-mark">W</span><span>WhatsNext</span></RouterLink
      >
      <div>
        <p class="eyebrow">你的 Personal Learning Agent</p>
        <h1>每一次学习，<br />都知道下一步。</h1>
        <p>从目标、资料到反馈，持续维护你的学习路径。</p>
      </div>
      <small>Goal → Plan → Action → Feedback → Replanning</small>
    </section>
    <section class="auth-form-wrap">
      <div class="auth-card"><slot /></div>
    </section>
  </div>
</template>
