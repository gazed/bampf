uniform float time;   // current time in seconds
uniform vec2  screen; // viewport size
uniform float alpha;  // transparency
out     vec4  ffc;    // final fragment colour

const float Pi     = 3.14159;
const float fScale = 4.3;
const float fEps   = 0.5;

void main()  {
   vec2 p = (2.0*gl_FragCoord.xy-screen)/max(screen.x,screen.y);
   for(int i=1; i<100; i++) {
       vec2 newp =p;
       newp.x += 1.5/float(i)*sin(float(i)*p.y+time/40.0+0.3*float(i))+400./20.0;
       newp.y += 0.05/float(i)*sin(float(i)*p.x+time/1.0+0.3*float(i+10))-400./20.0+15.0;
       p = newp;
   }
   vec3 col = vec3(0.5*sin(3.0*p.x)+0.5,0.5*sin(3.0*p.y)+0.5,sin(p.x+p.y));
   vec3 lum = vec3(0.299,0.587,0.114);
   vec3 c = vec3(dot(col*0.2,lum));
   ffc = vec4(c, 1.0);
   ffc.a = ffc.a*alpha;
}
