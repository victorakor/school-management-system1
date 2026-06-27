/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './web/templates/**/*.html',
    './web/static/js/**/*.js',
  ],
  theme: {
    extend: {
      colors: {
        primary:    '#0F2557',
        accent:     '#2563EB',
        gold:       '#F59E0B',
        surface:    '#FFFFFF',
        background: '#F1F5F9',
        'text-primary':   '#0F172A',
        'text-secondary': '#64748B',
        success:    '#10B981',
        warning:    '#F59E0B',
        danger:     '#F43F5E',
        neutral: {
          50:  '#F8FAFC',
          100: '#F1F5F9',
          200: '#E2E8F0',
          300: '#CBD5E1',
          400: '#94A3B8',
          500: '#64748B',
          600: '#475569',
          700: '#334155',
          800: '#1E293B',
          900: '#0F172A',
        },
      },
      fontFamily: {
        display: ['"Plus Jakarta Sans"', 'system-ui', 'sans-serif'],
        body:    ['Inter', 'system-ui', 'sans-serif'],
        mono:    ['"JetBrains Mono"', 'monospace'],
      },
      fontSize: {
        'display-xl': ['clamp(2rem, 5vw, 2.5rem)', { lineHeight: '1.1', letterSpacing: '-0.02em' }],
        'display-lg': ['clamp(1.5rem, 3.5vw, 1.875rem)', { lineHeight: '1.2' }],
        'display-md': ['clamp(1.25rem, 2.5vw, 1.5rem)', { lineHeight: '1.3' }],
        'body-lg':    ['1.125rem', { lineHeight: '1.75' }],
        'body-base':  ['1rem', { lineHeight: '1.6' }],
        'label':      ['0.875rem', { lineHeight: '1.4', letterSpacing: '0.05em' }],
        'caption':    ['0.75rem', { lineHeight: '1.4' }],
      },
      borderRadius: {
        'card': '16px',
        '2xl':  '16px',
        '3xl':  '24px',
      },
      boxShadow: {
        'card': '0 4px 24px rgba(0,0,0,0.06)',
        'card-hover': '0 8px 32px rgba(0,0,0,0.10)',
        'modal': '0 24px 64px rgba(0,0,0,0.15)',
      },
      spacing: {
        '18': '4.5rem',
        '22': '5.5rem',
        '88': '22rem',
        '112': '28rem',
        '128': '32rem',
      },
      transitionDuration: {
        'micro':      '100',
        'fast':       '200',
        'standard':   '300',
        'deliberate': '500',
      },
      animation: {
        'shimmer': 'shimmer 1.5s infinite',
        'pulse-badge': 'pulse-badge 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'marquee': 'marquee 30s linear infinite',
        'fade-in': 'fadeIn 300ms ease-out',
        'slide-up': 'slideUp 300ms ease-out',
      },
      keyframes: {
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        'pulse-badge': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.5' },
        },
        marquee: {
          '0%': { transform: 'translateX(0)' },
          '100%': { transform: 'translateX(-50%)' },
        },
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(16px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
      screens: {
        'xs': '360px',
        'sm': '640px',
        'md': '768px',
        'lg': '1024px',
        'xl': '1280px',
        '2xl': '1536px',
      },
    },
  },
  plugins: [],
};
