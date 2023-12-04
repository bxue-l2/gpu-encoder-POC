#include <stdio.h>
#include <cuda.h>

__global__ void vector_add(float *out, float *a, float *b, int n) {
    for(int i = 0; i < n; i++){
        out[i] = a[i] + b[i];
    }
}


extern "C" {
	void vectoradd(float *a, float *b, float *out, int N) {
		float *d_a, *d_b, *d_out;

		cudaMalloc((void**)&d_a, sizeof(float) * N);
		cudaMemcpy(d_a, a, sizeof(float) * N, cudaMemcpyHostToDevice);

		cudaMalloc((void**)&d_b, sizeof(float) * N);
		cudaMemcpy(d_b, b, sizeof(float) * N, cudaMemcpyHostToDevice);

		cudaMalloc((void**)&d_out, sizeof(float) * N);

		// Main function
		vector_add<<<1,1>>>(d_out, d_a, d_b, N);

		cudaMemcpy(out, d_out, sizeof(float) * N, cudaMemcpyDeviceToHost);

		cudaFree(d_a);
		cudaFree(d_b);
		cudaFree(d_out);
	}
}
