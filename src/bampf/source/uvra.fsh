#version 330

in      vec2      t_uv;
uniform sampler2D uv;
uniform float     fd;    // fade distance
uniform float     time;  // current time in seconds
uniform float     spin;  // rotation speed 0 -> 1
uniform float     alpha; // transparency
out     vec4      ffc;   // final fragment colour

float fade(float distance) {
   float z = gl_FragCoord.z / gl_FragCoord.w / distance;
   z = clamp(z, 0.0, 1.0);
   return 1.0 - z;
}
void main() {
   float sa = sin(time*spin);               // calculate rotation
   float ca = cos(time*spin);               // ..
   mat2 rot = mat2(ca, -sa, sa, ca);        // ..
   ffc = texture(uv, ((t_uv-0.5)*rot)+0.5); // rotate around its center
   ffc.a = ffc.a*fade(fd)*alpha;
}
