// one passive scroll listener; no smooth-scroll interception or animation loops.
const reduced=matchMedia('(prefers-reduced-motion: reduce)');
const progress=document.querySelector('.reading-progress');
let scheduled=false;
function update(){
 scheduled=false;
 const max=document.documentElement.scrollHeight-innerHeight;
 if(progress)progress.style.transform=`scaleX(${max>0?Math.min(1,scrollY/max):0})`;
}
addEventListener('scroll',()=>{if(!scheduled){scheduled=true;requestAnimationFrame(update)}},{passive:true});
addEventListener('resize',update);update();
if(!reduced.matches&&'IntersectionObserver' in window){
 const observer=new IntersectionObserver(entries=>entries.forEach(entry=>{
  if(!entry.isIntersecting)return;
  entry.target.animate([{opacity:.25,transform:'translateY(24px)'},{opacity:1,transform:'translateY(0)'}],{duration:750,easing:'cubic-bezier(.22,1,.36,1)'});
  observer.unobserve(entry.target);
 }),{threshold:.12});
 document.querySelectorAll('.section h2,.article-prose h2,.social-links,.article-heading h1,.strip-flow li').forEach(node=>observer.observe(node));
 reduced.addEventListener('change',()=>{if(reduced.matches){observer.disconnect();document.getAnimations().forEach(a=>a.cancel())}});
}
// article pages share navigation, but do not load the home carousel script.
if(document.body.classList.contains('article-page')){
 const menu=document.querySelector('.menu-toggle'),nav=document.querySelector('#mobile-nav'),header=document.querySelector('#site-header');
 function close(){nav.hidden=true;menu.setAttribute('aria-expanded','false');header.classList.remove('menu-open')}
 menu.addEventListener('click',()=>{const open=menu.getAttribute('aria-expanded')!=='true';nav.hidden=!open;menu.setAttribute('aria-expanded',String(open));header.classList.toggle('menu-open',open)});
 addEventListener('keydown',e=>{if(e.key==='Escape'){close();menu.focus()}});
 nav.addEventListener('click',e=>{if(e.target.closest('a'))close()});
 document.addEventListener('click',e=>{if(!header.contains(e.target))close()});
 matchMedia('(min-width:801px)').addEventListener('change',e=>{if(e.matches)close()});
}
