in      vec2      v_t;     // interpolated textured coordinates.
uniform sampler2D uv;
uniform float     fd;      // fade distance
uniform float     time;    // current time in seconds
uniform float     spin;    // rotation speed 0 -> 1
uniform float     alpha;   // transparency
out     vec4      f_color; // final fragment colour

float fade(float distance) {
   float z = gl_FragCoord.z / gl_FragCoord.w / distance;
   z = clamp(z, 0.0, 1.0);
   return 1.0 - z;
}
void main() {
   float sa = sin(time*spin);                  // calculate rotation
   float ca = cos(time*spin);                  // ..
   mat2 rot = mat2(ca, -sa, sa, ca);           // ..
   f_color = texture(uv, ((v_t-0.5)*rot)+0.5); // rotate around its center
   f_color.a = f_color.a*fade(fd)*alpha;
}
